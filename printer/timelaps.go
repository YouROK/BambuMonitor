package printer

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

type Timelaps struct {
	core          Core
	lastLayer     int
	lastTime      time.Time
	currentFolder string
	active        bool
	mu            sync.Mutex
	assembling    sync.Map
}

type TimelapsInfo struct {
	Name      string    `json:"name"`
	StartedAt time.Time `json:"started_at"`
	Status    string    `json:"status"` // "recording", "finished"
}

func NewTimelaps(core Core) *Timelaps {
	return &Timelaps{core: core}
}

func (t *Timelaps) Start() {
	t.active = true
	log.Println("[Timelapse] Мониторинг запущен")
	go t.worker()
}

func (t *Timelaps) worker() {
	ticker := time.NewTicker(time.Millisecond * time.Duration(t.core.GetConfig().Printer.EncodeWait))
	for t.active {
		<-ticker.C
		t.checkAndCapture()
	}
}

func (t *Timelaps) checkAndCapture() {
	cfg := t.core.GetConfig().Timelapse
	if !cfg.Enabled {
		if t.currentFolder != "" {
			t.currentFolder = ""
		}
		return
	}

	status := t.core.GetStatus()
	state, _ := status["gcode_state"].(string)

	if state != "RUNNING" {
		if t.currentFolder != "" {
			t.finalize()
		}
		return
	}

	// Извлекаем текущие данные
	currentLayer := 0
	if val, ok := status["layer_num"].(float64); ok {
		currentLayer = int(val)
	}

	shouldCapture := false
	captureSuffix := ""

	// ЛОГИКА ВЫБОРА РЕЖИМА
	if cfg.Interval == 0 {
		// Режим ПО СЛОЯМ
		if currentLayer > t.lastLayer {
			shouldCapture = true
		}
	} else {
		// Режим ПО ВРЕМЕНИ
		if time.Since(t.lastTime).Seconds() >= float64(cfg.Interval) {
			shouldCapture = true
		}
	}

	//TODO check extruder position in opencv

	if shouldCapture {
		t.lastTime = time.Now()
		t.lastLayer = currentLayer
		captureSuffix = fmt.Sprintf("layer_%04d_%d", currentLayer, time.Now().Unix())
		t.captureFrame(captureSuffix, status)
	}
}

func (t *Timelaps) captureFrame(suffix string, status map[string]any) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.currentFolder == "" {
		t.restoreOrCreateSession(status)
	}

	frame := t.core.GetFrame()
	if len(frame) == 0 {
		return
	}

	fileName := fmt.Sprintf("%s.jpg", suffix)
	filePath := filepath.Join(t.currentFolder, fileName)

	os.WriteFile(filePath, frame, 0644)
}

// Логика восстановления или создания новой папки
func (t *Timelaps) restoreOrCreateSession(status map[string]any) {
	taskName, _ := status["subtask_name"].(string)
	if taskName == "" {
		taskName = "unknown"
	}

	savePath := t.core.GetConfig().Timelapse.SavePath
	os.MkdirAll(savePath, 0755)

	entries, _ := os.ReadDir(savePath)
	for _, entry := range entries {
		if entry.IsDir() {
			infoPath := filepath.Join(savePath, entry.Name(), "info.json")
			if data, err := os.ReadFile(infoPath); err == nil {
				var info TimelapsInfo
				json.Unmarshal(data, &info)
				// Если имя совпадает и статус "recording" — подхватываем её
				if info.Name == taskName && info.Status == "recording" {
					t.currentFolder = filepath.Join(savePath, entry.Name())
					log.Printf("[Timelapse] Найдена активная сессия: %s", t.currentFolder)

					t.restoreLastLayer()
					return
				}
			}
		}
	}

	now := time.Now()
	baseName := fmt.Sprintf("%s_%02d_%02d", taskName, now.Month(), now.Day())

	num := 1
	for {
		folderName := fmt.Sprintf("%s_%d", baseName, num)
		fullPath := filepath.Join(savePath, folderName)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			t.currentFolder = fullPath
			break
		}
		num++
	}

	os.MkdirAll(t.currentFolder, 0755)
	info := TimelapsInfo{Name: taskName, StartedAt: now, Status: "recording"}
	infoData, _ := json.MarshalIndent(info, "", "  ")
	os.WriteFile(filepath.Join(t.currentFolder, "info.json"), infoData, 0644)
	log.Printf("[Timelapse] Новая сессия: %s", t.currentFolder)
}

// Вспомогательная функция для определения последнего сохраненного слоя
func (t *Timelaps) restoreLastLayer() {
	files, _ := filepath.Glob(filepath.Join(t.currentFolder, "layer_*.jpg"))
	if len(files) == 0 {
		return
	}
	sort.Strings(files)
	lastFile := files[len(files)-1]
	fmt.Sscanf(filepath.Base(lastFile), "layer_%d.jpg", &t.lastLayer)
}

func (t *Timelaps) finalize() {
	if t.currentFolder == "" {
		return
	}
	infoPath := filepath.Join(t.currentFolder, "info.json")
	if data, err := os.ReadFile(infoPath); err == nil {
		var info TimelapsInfo
		json.Unmarshal(data, &info)
		info.Status = "finished"
		newData, _ := json.MarshalIndent(info, "", "  ")
		os.WriteFile(infoPath, newData, 0644)

		log.Printf("[Timelapse] Печать завершена, папка %s", t.currentFolder)

		go func(folderName string) {
			log.Println("[Timelapse] Авто-сборка видео после печати...")
			err = t.AssembleVideo(folderName)
			if err != nil {
				log.Println("[Timelapse] Ошибка сборки видео:", err)
			}
		}(filepath.Base(t.currentFolder))
	}

	t.currentFolder = ""
	t.lastLayer = 0
}

func (t *Timelaps) AssembleVideo(folderName string) error {
	savePath := t.core.GetConfig().Timelapse.SavePath
	fullPath := filepath.Join(savePath, folderName)
	outputFile := filepath.Join(fullPath, "timelapse.mp4")

	if _, loading := t.assembling.LoadOrStore(folderName, true); loading {
		return fmt.Errorf("сборка этого видео уже запущена")
	}
	defer t.assembling.Delete(folderName)

	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return fmt.Errorf("папка не найдена")
	}

	fps := t.core.GetConfig().Timelapse.Fps
	if fps <= 0 {
		fps = 20
	}

	cmd := exec.Command("ffmpeg",
		"-y",
		"-framerate", fmt.Sprintf("%d", fps),
		"-pattern_type", "glob",
		"-i", filepath.Join(fullPath, "layer_*.jpg"),
		"-c:v", "libx264",
		"-pix_fmt", "yuv420p",
		"-profile:v", "high",
		"-level", "4.1",
		"-crf", "23",
		outputFile,
	)

	log.Printf("[Timelapse] Старт сборки: %s (FPS: %d)", folderName, fps)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("FFmpeg error: %v\noutput:\n%s", err, string(output))
	}

	t.markAsGenerated(fullPath)

	log.Printf("[Timelapse] Сборка завершена: %s", outputFile)
	return nil
}

func (t *Timelaps) markAsGenerated(folderPath string) {
	infoPath := filepath.Join(folderPath, "info.json")
	data, err := os.ReadFile(infoPath)
	if err != nil {
		return
	}

	var info TimelapsInfo
	json.Unmarshal(data, &info)
	info.Status = "finished"

	newData, _ := json.MarshalIndent(info, "", "  ")
	os.WriteFile(infoPath, newData, 0644)
}

func (t *Timelaps) Stop() {
	t.active = false
}
