package mqtt

func (m *BambuManager) getGCodeState() string {
	status := m.core.GetStatus()
	if state, ok := status["gcode_state"].(string); ok {
		return state
	}
	return "IDLE"
}
