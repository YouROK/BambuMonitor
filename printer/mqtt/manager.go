package mqtt

import (
	"bambucam/printer"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"strconv"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type BambuManager struct {
	core   printer.Core
	client mqtt.Client
	seqID  uint
}

func NewBambuManager(core printer.Core) *BambuManager {
	return &BambuManager{core: core}
}

func (m *BambuManager) Start() {
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

	opts.SetClientID("go-bambu-monitor-" + serial)
	opts.SetCleanSession(true)

	opts.SetAutoReconnect(true)
	opts.SetConnectRetry(true)
	opts.SetConnectRetryInterval(5 * time.Second)

	opts.OnConnect = func(c mqtt.Client) {
		log.Printf("[MQTT] Успешно подключено к %s", serial)

		topic := fmt.Sprintf("device/%s/report", serial)
		c.Subscribe(topic, 0, m.handleMessageStatus)

		m.RequestAllStatus()
	}
	opts.SetConnectionLostHandler(func(c mqtt.Client, err error) {
		log.Printf("[MQTT] Связь потеряна: %v. Ожидание восстановления...", err)
	})
	opts.SetReconnectingHandler(func(c mqtt.Client, options *mqtt.ClientOptions) {
		log.Println("[MQTT] Попытка повторного подключения к принтеру...")
	})

	m.client = mqtt.NewClient(opts)
	token := m.client.Connect()
	token.Wait()
	if token.Error() != nil {
		log.Printf("[MQTT] Ошибка подключения: %v", token.Error())
	}
}

func (m *BambuManager) Stop() {
	m.client.Disconnect(1000)
}

func (m *BambuManager) handleMessageStatus(client mqtt.Client, msg mqtt.Message) {
	var fullStatus map[string]interface{}
	buf := msg.Payload()
	if err := json.Unmarshal(buf, &fullStatus); err != nil {
		return
	}

	printData, ok := fullStatus["print"].(map[string]interface{})
	if !ok {
		return
	}
	m.core.UpdateStatus(printData)
}

func (m *BambuManager) sendPrintCommand(topic string, payload map[string]any) mqtt.Token {
	body, _ := json.Marshal(payload)
	var token mqtt.Token
	if m.client != nil && m.client.IsConnected() {
		token = m.client.Publish(topic, 0, false, body)
		log.Println("[MQTT] Команда отправлена:\n", string(body))
	}
	return token
}

func (m *BambuManager) getSequenceId() string {
	ret := strconv.FormatInt(int64(m.seqID), 10)
	m.seqID++
	if m.seqID > math.MaxUint {
		m.seqID = 0
	}
	return ret
}
