package web

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (s *Server) IndexHandler(c *gin.Context) {
	c.HTML(http.StatusOK, "index.go.html", gin.H{
		"Hostname":         s.core.GetConfig().Printer.Hostname,
		"TimelapseEnabled": s.core.GetConfig().Timelapse.Enabled,
		"WaitFrame":        s.core.GetConfig().Printer.EncodeWait,
		"Version":          s.core.GetAppVersion(),
	})
}
