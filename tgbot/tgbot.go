package tgbot

import (
	"bambucam/printer"
	"log"
	"time"

	tele "gopkg.in/telebot.v4"
	"gopkg.in/telebot.v4/middleware"
)

type Telegram struct {
	core printer.Core
	bot  *tele.Bot
}

func NewTelegram(core printer.Core) *Telegram {
	return &Telegram{core: core}
}

func (t *Telegram) Start() {
	cfg := t.core.GetConfig().Telegram
	if cfg.Token == "" {
		log.Println("[Telegram] Токен отсутствует")
		return
	}

	if len(cfg.AdminIds) == 0 {
		log.Println("[Telegram] Админы отсутствуют")
		return
	}

	pref := tele.Settings{
		Token:  cfg.Token,
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	}

	var err error

	bot, err := tele.NewBot(pref)
	if err != nil {
		log.Println("[Telegram] Ошибка запуска бота:", err)
		return
	}
	log.Println("[Telegram] Телеграм бот запустился")
	t.bot = bot

	bot.Use(middleware.Whitelist(cfg.AdminIds...))

	t.setupCommands(bot)

	go bot.Start()

	t.SendMessageAll("Bambu Monitor стартовал")
}

func (t *Telegram) Stop() {
	t.bot.Stop()
}

func (t *Telegram) SendMessageAll(message string) {
	admins := t.core.GetConfig().Telegram.AdminIds
	for _, adminID := range admins {
		recipient := &tele.User{ID: adminID}
		_, err := t.bot.Send(recipient, message)
		if err != nil {
			log.Printf("[Telegram] Ошибка отправки сообщения %d: %v\n", adminID, err)
		}
	}
}
