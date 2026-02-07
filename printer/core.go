package printer

import "bambucam/config"

type Core interface {
	Start()
	Restart()
	Stop()

	IsOnline() bool
	SetOnline(online bool)
	GetFrame() []byte
	UpdateFrame(frame []byte, fps float64)
	GetStatus() map[string]any
	UpdateStatus(status map[string]any)
	GetConfig() *config.Config
	SetConfig(cfg *config.Config)

	ToggleLight()

	AssembleVideo(folderName string) error
}
