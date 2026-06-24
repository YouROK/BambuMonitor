package tgbot

import (
	"bytes"
	"fmt"
	"strings"

	tele "gopkg.in/telebot.v4"
)

func (t *Telegram) setupCommands(bot *tele.Bot) {
	t.bot.Handle("/start", t.startBot)
	t.bot.Handle("/help", t.startBot)
	t.bot.Handle("/snap", t.sendSnap)
	t.bot.Handle("/status", t.sendStatus)
	t.bot.Handle("/light", t.toggleLight)
	t.bot.Handle("/timelapse", t.sendTimelapse)
	t.bot.Handle(&tele.InlineButton{Unique: "show_tl"}, t.handleTimelapseCallback)
}

func (t *Telegram) startBot(c tele.Context) error {
	return c.Send("Добро пожаловать в бот Bambu Monitor")
}

func (t *Telegram) sendHelp(c tele.Context) error {
	return nil
}

func (t *Telegram) sendSnap(c tele.Context) error {
	frame := t.core.GetFrame()
	if frame == nil {
		return c.Send("Кадр отсутствует")
	}
	photo := &tele.Photo{
		File: tele.FromReader(bytes.NewReader(frame)),
	}

	return c.Send(photo)
}

func (t *Telegram) toggleLight(c tele.Context) error {
	status := t.core.GetStatus()
	currentMode := "off"
	if report, ok := status["lights_report"].([]any); ok && len(report) > 0 {
		if light, ok := report[0].(map[string]any); ok {
			if mode, ok := light["mode"].(string); ok {
				currentMode = mode
			}
		}
	}

	t.core.ToggleLight()

	if currentMode == "on" {
		currentMode = "Выкл"
	} else {
		currentMode = "Вкл"
	}

	return c.Send("Свет: " + currentMode)
}

func (t *Telegram) sendStatus(c tele.Context) error {
	status := t.core.GetStatus()
	if status == nil {
		return c.Send("❌ Статус отсутствует")
	}

	// Проверка Online
	online, _ := status["online"].(bool)
	if !online {
		return c.Send("<b>🖨 Состояние: <pre>OFFLINE</pre></b>\nПринтер выключен или не в сети 🔌", tele.ModeHTML)
	}

	// Извлекаем основные данные с приведением типов
	state, _ := status["gcode_state"].(string)
	task, _ := status["subtask_name"].(string)
	nozzleTemp, _ := status["nozzle_temper"].(float64)
	bedTemp, _ := status["bed_temper"].(float64)
	percent, _ := status["mc_percent"].(float64)
	remaining, _ := status["mc_remaining_time"].(float64)
	wifi, _ := status["wifi_signal"].(string)

	var msg strings.Builder
	msg.WriteString("<b>Bambu Monitor</b>\n\n")
	msg.WriteString(fmt.Sprintf("📊 <b>Состояние:</b> %s\n", state))
	if task != "" {
		msg.WriteString(fmt.Sprintf("📝 <b>Задача:</b> %s\n", task))
	}
	msg.WriteString(" — — — — — — — — —\n")

	// Температуры
	msg.WriteString(fmt.Sprintf("🌡 <b>Сопло:</b> %.1f° | 🛏 <b>Стол:</b> %.1f°\n", nozzleTemp, bedTemp))

	// Прогресс
	if state != "IDLE" {
		msg.WriteString(fmt.Sprintf("\n<b>Прогресс: %.0f%%</b>\n", percent))
		msg.WriteString(fmt.Sprintf("⏳ Осталось: <b>%.0f мин</b>\n", remaining))
	}

	// Wi-Fi
	msg.WriteString(fmt.Sprintf("\n📶 <b>Wi-Fi:</b> %s\n", wifi))

	// Таймлапс
	timelapsEnable := t.core.GetConfig().Timelapse.Enabled
	statusTL := ""
	if timelapsEnable {
		statusTL = "Вкл"
	} else {
		statusTL = "Выкл"
	}
	msg.WriteString(fmt.Sprintf("🎬 <b>Таймлапсы:</b> %s\n", statusTL))

	// Подсветка
	lightStatus := "Выкл"
	if lights, ok := status["lights_report"].([]any); ok {
		for _, l := range lights {
			if m, ok := l.(map[string]any); ok {
				if m["node"] == "chamber_light" && m["mode"] == "on" {
					lightStatus = "Вкл"
				}
			}
		}
	}
	msg.WriteString(fmt.Sprintf("💡 <b>Подсветка:</b> %s\n", lightStatus))

	return c.Send(msg.String(), tele.ModeHTML)
}
