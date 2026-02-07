package web

import (
	"bambucam/printer"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"

	"github.com/gin-gonic/gin"
)

func (s *Server) HandleAssemble(c *gin.Context) {
	var req struct {
		Folder string `json:"folder"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Неверный запрос"})
		return
	}

	go func() {
		err := s.core.AssembleVideo(req.Folder)
		if err != nil {
			log.Printf("Ошибка сборки из веба: %v", err)
		}
	}()

	c.JSON(200, gin.H{"message": "Сборка запущена в фоновом режиме"})
}

func (s *Server) TimelapsHandler(c *gin.Context) {
	savePath := s.core.GetConfig().Timelapse.SavePath

	// Структура для передачи в шаблон
	type TimelapseView struct {
		FolderName string
		Name       string
		Date       string
		Status     string
		HasVideo   bool
		FrameCount int
		Thumbnail  string
	}

	var list []TimelapseView

	entries, _ := os.ReadDir(savePath)
	for _, entry := range entries {
		if entry.IsDir() {
			fullPath := filepath.Join(savePath, entry.Name())
			infoPath := filepath.Join(fullPath, "info.json")

			// Читаем info.json
			var info printer.TimelapsInfo
			data, err := os.ReadFile(infoPath)
			if err == nil {
				json.Unmarshal(data, &info)
			}

			// Считаем количество кадров
			frames, _ := filepath.Glob(filepath.Join(fullPath, "layer_*.jpg"))

			// Проверяем наличие видео
			_, videoErr := os.Stat(filepath.Join(fullPath, "timelapse.mp4"))

			view := TimelapseView{
				FolderName: entry.Name(),
				Name:       info.Name,
				Date:       info.StartedAt.Format("02.01.2006 15:04"),
				Status:     info.Status,
				HasVideo:   videoErr == nil,
				FrameCount: len(frames),
			}

			// Если есть хоть один кадр, используем его как превью
			if len(frames) > 0 {
				view.Thumbnail = filepath.Join(entry.Name(), filepath.Base(frames[len(frames)-1]))
			}

			list = append(list, view)
		}
	}

	sort.Slice(list, func(i, j int) bool {
		return list[i].Date > list[j].Date
	})

	c.HTML(http.StatusOK, "timelaps.go.html", gin.H{
		"Timelapses": list,
		"Config":     s.core.GetConfig(),
	})
}

func (s *Server) TimelapsThumbnail(c *gin.Context) {
	filePath := c.Param("path")
	filePath = filepath.Clean(filePath)
	savePath := s.core.GetConfig().Timelapse.SavePath
	fullPath := filepath.Join(savePath, filePath)

	info, err := os.Stat(fullPath)
	if err != nil || info.IsDir() {
		c.JSON(http.StatusNotFound, gin.H{"error": "Файл не найден"})
		return
	}

	c.File(fullPath)
}

func (s *Server) TimelapsVideo(c *gin.Context) {
	filePath := filepath.Clean(c.Param("path"))
	fullPath := filepath.Join(s.core.GetConfig().Timelapse.SavePath, filePath)
	c.File(fullPath)
}

func (s *Server) TimelapsRemove(c *gin.Context) {
	var req struct {
		Folder string `json:"folder"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный запрос"})
		return
	}

	filePath := filepath.Clean(req.Folder)
	fullPath := filepath.Join(s.core.GetConfig().Timelapse.SavePath, filePath)

	err := os.RemoveAll(fullPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}

	c.JSON(200, gin.H{"message": "Таймлапс удален"})
}
