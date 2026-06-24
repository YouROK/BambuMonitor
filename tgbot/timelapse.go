package tgbot

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	tele "gopkg.in/telebot.v4"
)

type TLMenuItem struct {
	FolderName string
	Name       string
	ModTime    time.Time
}

func (t *Telegram) sendTimelapse(c tele.Context) error {
	args := c.Args()

	if len(args) > 0 {
		return t.sendTimelapseByFolder(c, args[0])
	}

	return t.sendTimelapseMenu(c)
}

func (t *Telegram) handleTimelapseCallback(c tele.Context) error {
	folderName := c.Data()
	defer c.Respond()
	return t.sendTimelapseByFolder(c, folderName)
}

func (t *Telegram) sendTimelapseMenu(c tele.Context) error {
	cfg := t.core.GetConfig().Timelapse
	savePath := cfg.SavePath

	list, err := t.getTimelapseList(savePath)
	if err != nil || len(list) == 0 {
		return c.Send("❌ Нет собранных таймлапсов для просмотра.")
	}

	const chunkSize = 20
	chunks := chunkTimelapseList(list, chunkSize)

	for idx, chunk := range chunks {
		menu := &tele.ReplyMarkup{}
		var rows []tele.Row

		for _, tl := range chunk {
			btnText := tl.Name
			if len(btnText) > 28 {
				btnText = btnText[:25] + "..."
			}

			btn := menu.Data(btnText, "show_tl", tl.FolderName)
			rows = append(rows, menu.Row(btn))
		}
		menu.Inline(rows...)

		title := "🎬 <b>Выберите таймлапс для просмотра:</b>"
		if len(chunks) > 1 {
			title = fmt.Sprintf("🎬 <b>Выберите таймлапс для просмотра (Часть %d из %d):</b>", idx+1, len(chunks))
		}

		if err := c.Send(title, menu, tele.ModeHTML); err != nil {
			return err
		}
	}

	return nil
}

func (t *Telegram) sendTimelapseByFolder(c tele.Context, folderName string) error {
	cfg := t.core.GetConfig().Timelapse
	savePath := cfg.SavePath

	fullPath := filepath.Join(savePath, folderName)
	mp4Path := filepath.Join(fullPath, "timelapse.mp4")
	previewPath := filepath.Join(fullPath, "preview.mp4")
	infoPath := filepath.Join(fullPath, "info.json")

	mp4St, err := os.Stat(mp4Path)
	if os.IsNotExist(err) {
		return c.Send(fmt.Sprintf("❌ Видеофайл в папке %s отсутствует или еще не собран.", folderName))
	} else if err != nil {
		return c.Send("❌ Ошибка при чтении файла: " + err.Error())
	}

	var info struct {
		Name string `json:"name"`
	}
	if data, err := os.ReadFile(infoPath); err == nil {
		_ = json.Unmarshal(data, &info)
	}
	if info.Name == "" {
		info.Name = folderName
	}

	serverHost := t.core.GetConfig().Web.Hostname

	if !strings.HasPrefix(serverHost, "http://") && !strings.HasPrefix(serverHost, "https://") {
		serverHost = "http://" + serverHost
	}

	downloadURL := fmt.Sprintf("%s/tl/file/%s/timelapse.mp4", serverHost, folderName)

	const maxTelegramSize = 50 * 1024 * 1024 // 50 MB
	mp4Size := mp4St.Size()

	if mp4Size <= maxTelegramSize {
		caption := fmt.Sprintf("🎬 <b>Таймлапс:</b> %s\n📦 <b>Размер:</b> %s\n\n🔗 <a href=\"%s\">Скачать напрямую</a>",
			info.Name,
			humanize.Bytes(uint64(mp4Size)),
			downloadURL,
		)

		_ = c.Send("⏳ Отправляю оригинальный видеофайл...")
		video := &tele.Video{
			File:    tele.FromDisk(mp4Path),
			Caption: caption,
		}
		return c.Send(video, tele.ModeHTML)
	}

	if previewSt, err := os.Stat(previewPath); err == nil {
		previewSize := previewSt.Size()
		if previewSize <= maxTelegramSize {
			previewCaption := fmt.Sprintf("🎬 <b>Таймлапс:</b> %s (Превью)\n⚠️ <i>Оригинал слишком большой (%s) для отправки напрямую.</i>\n\n🔗 <a href=\"%s\">Скачать оригинал в полном качестве</a>",
				info.Name,
				humanize.Bytes(uint64(mp4Size)),
				downloadURL,
			)

			_ = c.Send("⏳ Отправляю сжатое превью...")
			video := &tele.Video{
				File:    tele.FromDisk(previewPath),
				Caption: previewCaption,
			}
			return c.Send(video, tele.ModeHTML)
		}
	}

	text := fmt.Sprintf("🎬 <b>Таймлапс:</b> %s\n❌ <i>Файл слишком большой (%s) для отправки в Telegram, а превью отсутствует.</i>\n\n🔗 <a href=\"%s\">Скачать оригинальное видео напрямую</a>",
		info.Name,
		humanize.Bytes(uint64(mp4Size)),
		downloadURL,
	)
	return c.Send(text, tele.ModeHTML)
}

func (t *Telegram) getTimelapseList(savePath string) ([]TLMenuItem, error) {
	entries, err := os.ReadDir(savePath)
	if err != nil {
		return nil, err
	}

	var list []TLMenuItem
	for _, entry := range entries {
		if entry.IsDir() {
			folderName := entry.Name()
			fullPath := filepath.Join(savePath, folderName)
			mp4Path := filepath.Join(fullPath, "timelapse.mp4")
			infoPath := filepath.Join(fullPath, "info.json")

			if _, err := os.Stat(mp4Path); err != nil {
				continue
			}

			var info struct {
				Name string `json:"name"`
			}
			if data, err := os.ReadFile(infoPath); err == nil {
				_ = json.Unmarshal(data, &info)
			}
			if info.Name == "" {
				info.Name = folderName
			}

			infoEntry, err := entry.Info()
			var modTime time.Time
			if err == nil {
				modTime = infoEntry.ModTime()
			}

			list = append(list, TLMenuItem{
				FolderName: folderName,
				Name:       info.Name,
				ModTime:    modTime,
			})
		}
	}

	sort.Slice(list, func(i, j int) bool {
		return list[i].ModTime.After(list[j].ModTime)
	})

	return list, nil
}

func chunkTimelapseList(slice []TLMenuItem, chunkSize int) [][]TLMenuItem {
	var chunks [][]TLMenuItem
	for i := 0; i < len(slice); i += chunkSize {
		end := i + chunkSize
		if end > len(slice) {
			end = len(slice)
		}
		chunks = append(chunks, slice[i:end])
	}
	return chunks
}
