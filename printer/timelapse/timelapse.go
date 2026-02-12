package timelapse

import (
	"bambucam/printer"
	"log"
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
}

func (t *Timelapse) Stop() {
	close(t.stop)
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
