package timelapse

import "time"

type TimelapsInfo struct {
	Name      string
	StartedAt time.Time
	Status    TLStatus `json:"status"`
}

type TLStatus int

const (
	TL_IDLE TLStatus = iota
	TL_RECORDING
	TL_PAUSED
	TL_CONVERT
	TL_ERROR
	TL_FINISHED
)

func (s TLStatus) String() string {
	switch s {
	case TL_IDLE:
		return "idle"
	case TL_RECORDING:
		return "recording"
	case TL_PAUSED:
		return "paused"
	case TL_CONVERT:
		return "convert"
	case TL_ERROR:
		return "error"
	case TL_FINISHED:
		return "finished"
	default:
		return "unknown"
	}
}
