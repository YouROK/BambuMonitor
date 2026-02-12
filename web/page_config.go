package web

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

func (s *Server) ConfigHandler(c *gin.Context) {
	c.HTML(http.StatusOK, "config.go.html", gin.H{
		"Config": s.core.GetConfig(),
	})
}

func (s *Server) ConfigSetter(c *gin.Context) {
	cfg := s.core.GetConfig()

	// Принтер
	cfg.Printer.Hostname = c.PostForm("printer_hostname")
	cfg.Printer.Password = c.PostForm("printer_password")
	cfg.Printer.Serial = c.PostForm("printer_serial")
	if val, err := strconv.Atoi(c.PostForm("printer_encode_wait")); err == nil {
		cfg.Printer.EncodeWait = val
	}

	// Веб
	cfg.Web.BindAddress = c.PostForm("web_address")
	if val, err := strconv.Atoi(c.PostForm("web_port")); err == nil {
		cfg.Web.Port = val
	}
	cfg.Web.Username = c.PostForm("web_username")
	cfg.Web.Password = c.PostForm("web_password")

	// Таймлапс
	// Чекбоксы в HTML приходят как "on", если включены, или отсутствуют вовсе
	cfg.Timelapse.Enabled = c.PostForm("tl_enabled") == "on"
	cfg.Timelapse.SavePath = c.PostForm("tl_path")
	if val, err := strconv.Atoi(c.PostForm("tl_fps")); err == nil {
		cfg.Timelapse.Fps = val
	}
	if val, err := strconv.Atoi(c.PostForm("tl_after_layer")); err == nil {
		cfg.Timelapse.AfterLayer = val
	}
	if val, err := strconv.Atoi(c.PostForm("tl_interval")); err == nil {
		cfg.Timelapse.Interval = val
	}
	cfg.Timelapse.AddTime = c.PostForm("tl_addtime") == "on"

	// Сохраняем и обновляем в памяти
	s.core.SetConfig(cfg)
	go func() {
		time.Sleep(time.Second * 5)
		s.core.Restart()
	}()

	// Возвращаемся на главную или показываем сообщение об успехе
	c.Redirect(http.StatusSeeOther, "/")
}
