package web

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

//func (s *Server) StreamHandler(c *gin.Context) {
//	if !s.core.IsOnline() {
//		c.AbortWithStatus(http.StatusServiceUnavailable)
//		return
//	}
//	w := c.Writer
//	header := w.Header()
//	header.Set("Content-Type", "multipart/x-mixed-replace; boundary=boundarydonotcross")
//	header.Set("Cache-Control", "no-cache")
//	header.Set("Connection", "keep-alive")
//
//	for {
//		select {
//		case <-c.Request.Context().Done():
//			log.Println("[Web] Клиент отключился, завершаем стрим")
//			return
//		default:
//			frame := s.core.GetFrame()
//			if frame == nil {
//				log.Println("[Web] Кадров нет, закрываем соединение для перезагрузки клиентом")
//				return
//			}
//
//			_, err := w.Write([]byte("--boundarydonotcross\r\nContent-Type: image/jpeg\r\n\r\n"))
//			if err != nil {
//				return
//			}
//
//			_, err = w.Write(frame)
//			if err != nil {
//				return
//			}
//
//			_, err = w.Write([]byte("\r\n"))
//			if err != nil {
//				return
//			}
//
//			w.(http.Flusher).Flush()
//			time.Sleep(time.Millisecond * time.Duration(s.core.GetConfig().Printer.EncodeWait))
//		}
//	}
//}

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
