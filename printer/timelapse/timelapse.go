package timelapse

import (
	"bambucam/printer"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Timelapse struct {
	core printer.Core

	status     TLStatus
	stop       chan struct{}
	mu         sync.Mutex
	assembling sync.Map

	lastLayer     int
	lastTime      time.Time
	startTime     time.Time
	currentFolder string
	currentTask   string
}

func NewTimelapse(core printer.Core) *Timelapse {
	return &Timelapse{
		core: core,
		stop: make(chan struct{}),
	}
}

func (t *Timelapse) Start() {
	log.Println("[Timelapse] Мониторинг запущен")
	go t.worker()
	go t.generateMissingPreviews()
}

func (t *Timelapse) Stop() {
	if t != nil && t.stop != nil {
		close(t.stop)
	}
}

func (t *Timelapse) worker() {
	ticker := time.NewTicker(time.Millisecond * time.Duration(t.core.GetConfig().Printer.EncodeWait))
	for {
		select {
		case <-t.stop:
			return
		case <-ticker.C:
			t.checkTimelapse()
		}
	}
}

func (t *Timelapse) generateMissingPreviews() {
	savePath := t.core.GetConfig().Timelapse.SavePath
	if savePath == "" {
		log.Println("[Timelapse] Ошибка авто-сканирования: путь сохранения пуст")
		return
	}

	entries, err := os.ReadDir(savePath)
	if err != nil {
		log.Printf("[Timelapse] Ошибка чтения директории при авто-сканировании: %v", err)
		return
	}

	log.Println("[Timelapse] Запуск фонового сканирования пропущенных превью...")
	count := 0

	for _, entry := range entries {
		if entry.IsDir() {
			folderName := entry.Name()
			fullPath := filepath.Join(savePath, folderName)

			mp4Path := filepath.Join(fullPath, "timelapse.mp4")
			previewPath := filepath.Join(fullPath, "preview.mp4")

			if _, err := os.Stat(mp4Path); err == nil {
				if _, err := os.Stat(previewPath); os.IsNotExist(err) {
					log.Printf("[Timelapse] Найдено видео без превью в папке: %s. Начинаю сборку...", folderName)

					if err := t.AssemblePreview(folderName); err != nil {
						log.Printf("[Timelapse] Не удалось сгенерировать превью для %s: %v", folderName, err)
					} else {
						count++
					}
				}
			}
		}
	}

	if count > 0 {
		log.Printf("[Timelapse] Фоновое сканирование завершено. Сгенерировано новых превью: %d", count)
	}
}
