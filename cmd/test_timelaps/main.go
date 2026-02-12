package main

import (
	"bambucam/config"
	"bambucam/printer"
	"bambucam/printer/timelapse"
	"log"
	"os"
	"sync"
	"time"
)

type MockApp struct {
	cfg       *config.Config
	lastFrame []byte
	status    map[string]any

	configMutex sync.RWMutex
	frameMutex  sync.RWMutex
	statusMutex sync.RWMutex

	bambucam  *printer.BambuCamera
	timelapse *timelapse.Timelapse
}

func (a *MockApp) GetFrame() []byte {
	a.frameMutex.RLock()
	defer a.frameMutex.RUnlock()
	return a.lastFrame
}

func (a *MockApp) UpdateFrame(frame []byte, fps float64) {
	a.frameMutex.Lock()
	defer a.frameMutex.Unlock()
	a.lastFrame = frame
}

func (a *MockApp) GetStatus() map[string]any {
	a.statusMutex.RLock()
	defer a.statusMutex.RUnlock()
	return a.status
}

func (a *MockApp) UpdateStatus(status map[string]any) {
	a.statusMutex.Lock()
	defer a.statusMutex.Unlock()
	for key, val := range status {
		a.status[key] = val
	}
}

func (a *MockApp) GetConfig() *config.Config {
	a.configMutex.RLock()
	defer a.configMutex.RUnlock()
	return a.cfg
}

func (a *MockApp) SetConfig(cfg *config.Config) {
	a.configMutex.Lock()
	defer a.configMutex.Unlock()
	a.cfg = cfg
	err := a.cfg.Save()
	if err != nil {
		log.Printf("Failed to save config: %v", err)
	}
}

func (a *MockApp) ToggleLight()  {}
func (a *MockApp) StopPrinting() {}
func (a *MockApp) TogglePause()  {}
func (a *MockApp) IsOnline() bool {
	return true
}
func (a *MockApp) SetOnline(online bool) {}
func (a *MockApp) GetAppVersion() string {
	return "Test timelapse"
}

func (a *MockApp) AssembleVideo(folderName string) error {
	return a.timelapse.AssembleVideo(folderName)
}

func (a *MockApp) Run() {
	a.Start()
	time.Sleep(2 * time.Second)
	log.Println("--- Симуляция печати началась ---")

	a.UpdateStatus(map[string]any{
		"gcode_state":  "RUNNING",
		"subtask_name": "TestTimelapse",
		"layer_num":    1.0,
	})

	time.Sleep(3 * time.Second)

	for i := 0; i < 10; i++ {
		a.UpdateStatus(map[string]any{"layer_num": float64(i)})
		log.Println("Слой", i, "установлен")
		time.Sleep(1 * time.Second)
	}

	a.UpdateStatus(map[string]any{"gcode_state": "FINISH"})
	log.Println("Печать завершена")

	time.Sleep(5 * time.Second)

	a.Stop()

	log.Println("Тест окончен ./timelapse")
}

func (a *MockApp) Start() {
	a.status = make(map[string]any)

	a.bambucam = printer.NewBambuCamera(a)
	a.bambucam.Start()

	a.timelapse = timelapse.NewTimelapse(a)
	a.timelapse.Start()
}

func (a *MockApp) Restart() {
	a.Stop()
	a.Start()
}

func (a *MockApp) Stop() {
	a.bambucam.Stop()
	a.timelapse.Stop()
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Println("Error loading config:", err)
		os.Exit(1)
	}

	// 1. Инициализация конфига
	cfg.Timelapse.Enabled = true
	cfg.Timelapse.Interval = 1
	cfg.Timelapse.Fps = 5
	cfg.Timelapse.SavePath = "./timelapse"

	mock := &MockApp{
		cfg: cfg,
	}

	mock.Run()
}
