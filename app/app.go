package app

import (
	"bambucam/config"
	"bambucam/printer"
	"bambucam/web"
	"log"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
)

type App struct {
	cfg       *config.Config
	lastFrame []byte
	fps       float64
	status    sync.Map
	online    atomic.Bool

	configMutex sync.RWMutex
	frameMutex  sync.RWMutex

	webserver *web.Server
	bambustat *printer.BambuStatus
	bambucam  *printer.BambuCamera
	timelaps  *printer.Timelaps
}

func New() *App {
	a := &App{}

	var err error
	a.cfg, err = config.Load()
	if err != nil {
		log.Println("Error loading config:", err)
		os.Exit(1)
	}
	a.SetOnline(false)

	return a
}

func (a *App) IsOnline() bool {
	return a.online.Load()
}

func (a *App) SetOnline(online bool) {
	a.online.Store(online)
}

func (a *App) GetFrame() []byte {
	a.frameMutex.RLock()
	defer a.frameMutex.RUnlock()
	return a.lastFrame
}

func (a *App) UpdateFrame(frame []byte, fps float64) {
	a.frameMutex.Lock()
	defer a.frameMutex.Unlock()
	a.lastFrame = frame
	a.fps = fps
}

func (a *App) GetStatus() map[string]any {
	a.status.Store("fps", a.fps)
	a.status.Store("online", a.online.Load())
	normalMap := make(map[string]any)

	a.status.Range(func(key, value any) bool {
		normalMap[key.(string)] = value
		return true
	})
	return normalMap
}

func (a *App) UpdateStatus(status map[string]any) {
	for key, val := range status {
		a.status.Store(key, val)
	}
}

func (a *App) GetConfig() *config.Config {
	a.configMutex.RLock()
	defer a.configMutex.RUnlock()
	return a.cfg
}

func (a *App) SetConfig(cfg *config.Config) {
	a.configMutex.Lock()
	defer a.configMutex.Unlock()
	a.cfg = cfg
	err := a.cfg.Save()
	if err != nil {
		log.Printf("Failed to save config: %v", err)
	}
}

func (a *App) ToggleLight() {
	if a.bambustat != nil {
		a.bambustat.ToggleLight()
	}
}

func (a *App) AssembleVideo(folderName string) error {
	return a.timelaps.AssembleVideo(folderName)
}

func (a *App) Run() {
	a.Start()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("Завершение работы...")
	a.Stop()
}

func (a *App) Start() {
	var err error
	a.cfg, err = config.Load()
	if err != nil {
		log.Println("Error loading config:", err)
		os.Exit(1)
	}

	a.bambucam = printer.NewBambuCamera(a)
	a.bambucam.Start()

	a.bambustat = printer.NewMqttClient(a)
	a.bambustat.Start()

	a.timelaps = printer.NewTimelaps(a)
	a.timelaps.Start()

	a.webserver = web.NewServer(a)
	a.webserver.Start()
}

func (a *App) Restart() {
	a.Stop()
	a.Start()
}

func (a *App) Stop() {
	a.bambucam.Stop()
	a.bambustat.Stop()
	a.webserver.Stop()
	a.timelaps.Stop()
}
