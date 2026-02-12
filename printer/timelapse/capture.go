package timelapse

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

func (t *Timelapse) checkTimelapse() {
	cfg := t.core.GetConfig().Timelapse
	if !cfg.Enabled {
		return
	}

	state, _ := t.core.GetStatus()["gcode_state"].(string)
	status := t.status

	switch state {
	case "RUNNING":
		if status == TL_IDLE {
			t.startCapture()
		}
		t.captureIfNeeded()

	case "PAUSED":
		if status == TL_RECORDING {
			t.pause()
		}

	default: // FINISHED, STOPPED, ERROR
		if status == TL_RECORDING || status == TL_PAUSED {
			t.finalize()
		}
	}
}

func (t *Timelapse) startCapture() {
	// Принтер начал печать
	if t.status != TL_RECORDING && t.status != TL_PAUSED {
		savePath := t.core.GetConfig().Timelapse.SavePath
		os.MkdirAll(savePath, 0755)

		taskName, _ := t.core.GetStatus()["subtask_name"].(string)
		if taskName == "" {
			taskName = "unknown"
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

		if t.currentFolder == "" {
			log.Println("[Timelapse] Ошибка при создании директории для таймлапса")
			return
		}

		os.MkdirAll(t.currentFolder, 0755)

		t.status = TL_RECORDING
		t.currentTask = taskName
		t.startTime = time.Now()
		t.saveStatus()
		log.Printf("[Timelapse] Новая сессия: %s", t.currentFolder)
	} else if t.status == TL_PAUSED {
		t.status = TL_RECORDING
		t.saveStatus()
	}
}

func (t *Timelapse) captureIfNeeded() {
	if t.status == TL_RECORDING {
		t.mu.Lock()
		defer t.mu.Unlock()

		currentLayer := 0
		if val, ok := t.core.GetStatus()["layer_num"].(float64); ok {
			currentLayer = int(val)
		}
		cfg := t.core.GetConfig().Timelapse

		shouldCapture := false

		// Снимать только после указанного слоя
		if currentLayer > cfg.AfterLayer {
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
		}

		if shouldCapture {
			t.lastTime = time.Now()
			t.lastLayer = currentLayer
			captureSuffix := fmt.Sprintf("layer_%04d_%d", currentLayer, time.Now().Unix())

			frame := t.core.GetFrame()
			if len(frame) == 0 {
				return
			}

			fileName := fmt.Sprintf("%s.jpg", captureSuffix)
			filePath := filepath.Join(t.currentFolder, fileName)

			buf, err := AddTimestampWithRoundedBox(frame, time.Now())
			if err == nil {
				frame = buf
			}

			os.WriteFile(filePath, frame, 0644)
		}
	}
}

func (t *Timelapse) pause() {
	if t.status == TL_RECORDING {
		t.status = TL_PAUSED
		t.saveStatus()
		log.Println("[Timelapse] Пауза записи")
	}
}

func (t *Timelapse) finalize() {
	t.status = TL_CONVERT
	t.saveStatus()
	log.Printf("[Timelapse] Печать завершена, папка %s", t.currentFolder)
	go func(folderName string) {
		log.Println("[Timelapse] Авто-сборка видео после печати...")
		err := t.AssembleVideo(folderName)
		if err != nil {
			t.status = TL_ERROR
			t.saveStatus()
			log.Println("[Timelapse] Ошибка сборки видео:", err)
		} else {
			t.status = TL_FINISHED
			t.saveStatus()
		}
		t.status = TL_IDLE
		t.currentTask = ""
		t.currentFolder = ""
	}(filepath.Base(t.currentFolder))
}

func (t *Timelapse) saveStatus() {
	info := TimelapsInfo{
		Name:      t.currentTask,
		Status:    t.status,
		StartedAt: t.startTime,
	}
	infoData, _ := json.MarshalIndent(info, "", " ")
	os.WriteFile(filepath.Join(t.currentFolder, "info.json"), infoData, 0644)
}
