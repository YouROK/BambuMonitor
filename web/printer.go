package web

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (s *Server) SnapHandler(c *gin.Context) {
	frame := s.core.GetFrame()
	if frame == nil {
		c.Status(404)
		return
	}
	c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
	c.Header("Pragma", "no-cache")
	c.Header("Expires", "0")

	c.Data(200, "image/jpeg", frame)
}

func (s *Server) PrinterStatus(c *gin.Context) {
	c.JSON(http.StatusOK, s.core.GetStatus())
}

func (s *Server) ToggleLight(c *gin.Context) {
	s.core.ToggleLight()
}

func (s *Server) StopPrinting(c *gin.Context) {
	s.core.StopPrinting()
}

func (s *Server) TogglePause(c *gin.Context) {
	s.core.TogglePause()
}
