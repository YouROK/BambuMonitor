package web

import (
	"bambucam/printer"
	"bambucam/web/static"
	"context"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type Server struct {
	core       printer.Core
	Router     *gin.Engine
	httpServer *http.Server
}

func NewServer(core printer.Core) *Server {
	//gin.SetMode(gin.ReleaseMode)

	r := gin.New()
	r.Use(gin.Recovery())

	s := &Server{
		core:   core,
		Router: r,
	}

	static.RouteEmbedFiles(r)
	s.SetupRouts()
	return s
}

func (s *Server) Start() {
	addr := s.core.GetConfig().Web.BindAddress + ":" + strconv.Itoa(s.core.GetConfig().Web.Port)

	s.httpServer = &http.Server{
		Addr:    addr,
		Handler: s.Router,
	}

	log.Printf("[WEB] Сервер запускается на %s", addr)

	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Println("[WEB] Ошибка запуска:", err)
		}
	}()
}

func (s *Server) Stop() {
	log.Println("[WEB] Останавливаю сервер...")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if s.httpServer != nil {
		err := s.httpServer.Shutdown(ctx)
		if err != nil {
			s.httpServer.Close()
		}
	}
}

func (s *Server) SetupRouts() {
	var route *gin.RouterGroup
	if s.core.GetConfig().Web.Username != "" && s.core.GetConfig().Web.Username != "" {
		route = s.Router.Group("/", gin.BasicAuth(gin.Accounts{
			s.core.GetConfig().Web.Username: s.core.GetConfig().Web.Password,
		}))
	} else {
		route = s.Router.Group("/")
	}
	route.GET("/", s.IndexHandler)
	route.GET("/config", s.ConfigHandler)
	route.GET("/status", s.PrinterStatus)
	route.GET("/timelapse", s.TimelapsHandler)
	route.GET("/tl/thumb/*path", s.TimelapsThumbnail)
	route.GET("/tl/video/*path", s.TimelapsVideo)
	route.GET("/snap", s.SnapHandler)

	route.POST("/config", s.ConfigSetter)
	route.POST("/printer/light", s.ToggleLight)
	route.POST("/printer/stop", s.StopPrinting)
	route.POST("/printer/pause", s.TogglePause)
	route.POST("/assemblevideo", s.HandleAssemble)
	route.POST("/tl/remove", s.TimelapsRemove)
}
