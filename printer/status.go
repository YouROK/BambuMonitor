package printer

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type BambuStatus struct {
	core   Core
	client mqtt.Client
}

func NewMqttClient(core Core) *BambuStatus {
	return &BambuStatus{core: core}
}

func (m *BambuStatus) Start() {
	serial := m.core.GetConfig().Printer.Serial
	if serial == "" {
		log.Println("[MQTT] Ошибка: Серийный номер не указан в конфиге!")
		return
	}

	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tls://%s:8883", m.core.GetConfig().Printer.Hostname))
	opts.SetUsername("bblp")
	opts.SetPassword(m.core.GetConfig().Printer.Password)
	opts.SetTLSConfig(&tls.Config{InsecureSkipVerify: true})

	// Уникальный ID важен, чтобы принтер не разрывал соединение
	opts.SetClientID("go-bambu-monitor-" + serial)
	opts.SetCleanSession(true)

	opts.OnConnect = func(c mqtt.Client) {
		log.Printf("[MQTT] Успешно подключено к %s", serial)

		reportTopic := fmt.Sprintf("device/%s/report", serial)
		c.Subscribe(reportTopic, 0, m.handleMessage)

		requestTopic := fmt.Sprintf("device/%s/request", serial)
		payload := `{"pushing": {"sequence_id": "0", "command": "pushall"}}`

		token := c.Publish(requestTopic, 0, false, payload)
		token.Wait()
		log.Println("[MQTT] Запрос pushall отправлен")
	}

	m.client = mqtt.NewClient(opts)
	if token := m.client.Connect(); token.Wait() && token.Error() != nil {
		log.Printf("[MQTT] Ошибка подключения: %v", token.Error())
		return
	}
}

func (m *BambuStatus) Stop() {
	m.client.Disconnect(1000)
}

func (m *BambuStatus) handleMessage(client mqtt.Client, msg mqtt.Message) {
	var fullStatus map[string]interface{}
	if err := json.Unmarshal(msg.Payload(), &fullStatus); err != nil {
		return
	}

	// Извлекаем вложенный объект "print"
	printData, ok := fullStatus["print"].(map[string]interface{})
	if !ok {
		return
	}
	// Обновление идет добавлением полей
	m.core.UpdateStatus(printData)
}

func (m *BambuStatus) ToggleLight() {
	status := m.core.GetStatus()

	// 2. Пытаемся понять, горит ли свет сейчас
	currentMode := "off"
	if report, ok := status["lights_report"].([]any); ok && len(report) > 0 {
		if light, ok := report[0].(map[string]any); ok {
			if mode, ok := light["mode"].(string); ok {
				currentMode = mode
			}
		}
	}

	// 3. Инвертируем состояние
	newMode := "on"
	if currentMode == "on" {
		newMode = "off"
	}

	// 4. Формируем JSON команду для Bambu Lab
	topic := fmt.Sprintf("device/%s/request", m.core.GetConfig().Printer.Serial)
	payload := map[string]any{
		"system": map[string]any{
			"sequence_id":   "2000",
			"command":       "ledctrl",
			"led_node":      "chamber_light",
			"led_mode":      newMode,
			"led_on_time":   500,
			"led_off_time":  500,
			"loop_times":    0,
			"interval_time": 0,
		},
	}

	body, _ := json.Marshal(payload)

	// 5. Публикуем в MQTT
	if m.client != nil && m.client.IsConnected() {
		m.client.Publish(topic, 0, false, body)
		log.Printf("[MQTT] Light toggled to: %s", newMode)
	}
}

func getFloat(v interface{}) float64 {
	if f, ok := v.(float64); ok {
		return f
	}
	return 0
}
