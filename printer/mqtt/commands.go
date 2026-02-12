package mqtt

import (
	"fmt"
	"log"
)

func (m *BambuManager) RequestAllStatus() {
	serial := m.core.GetConfig().Printer.Serial
	topic := fmt.Sprintf("device/%s/request", serial)
	payload := map[string]any{
		"pushing": map[string]any{
			"sequence_id": m.getSequenceId(),
			"command":     "pushall",
		},
	}

	token := m.sendPrintCommand(topic, payload)
	token.Wait()
}

func (m *BambuManager) ToggleLight() {
	status := m.core.GetStatus()

	currentMode := "off"
	if report, ok := status["lights_report"].([]any); ok && len(report) > 0 {
		if light, ok := report[0].(map[string]any); ok {
			if mode, ok := light["mode"].(string); ok {
				currentMode = mode
			}
		}
	}

	newMode := "on"
	if currentMode == "on" {
		newMode = "off"
	}

	topic := fmt.Sprintf("device/%s/request", m.core.GetConfig().Printer.Serial)
	payload := map[string]any{
		"system": map[string]any{
			"sequence_id":   m.getSequenceId(),
			"command":       "ledctrl",
			"led_node":      "chamber_light",
			"led_mode":      newMode,
			"led_on_time":   500,
			"led_off_time":  500,
			"loop_times":    0,
			"interval_time": 0,
		},
	}

	m.sendPrintCommand(topic, payload)
}

func (m *BambuManager) StopPrinting() {
	currentState := m.getGCodeState()

	if currentState == "IDLE" || currentState == "FINISH" || currentState == "FAILED" {
		log.Printf("[MQTT] StopPrinting: Принтер не печатает (state: %s)", currentState)
		return
	}

	topic := fmt.Sprintf("device/%s/request", m.core.GetConfig().Printer.Serial)
	payload := map[string]any{
		"print": map[string]any{
			"sequence_id": m.getSequenceId(),
			"command":     "stop",
			"param":       "",
		},
	}

	m.sendPrintCommand(topic, payload)
}

func (m *BambuManager) TogglePause() {
	currentState := m.getGCodeState()
	var command string

	switch currentState {
	case "RUNNING", "PREPARE":
		command = "pause"
	case "PAUSE":
		command = "resume"
	default:
		log.Printf("[MQTT] TogglePause: Принтер не печатает (state: %s)", currentState)
		return
	}

	topic := fmt.Sprintf("device/%s/request", m.core.GetConfig().Printer.Serial)
	payload := map[string]any{
		"print": map[string]any{
			"sequence_id": m.getSequenceId(),
			"command":     command,
			"param":       "",
		},
	}

	m.sendPrintCommand(topic, payload)
}
