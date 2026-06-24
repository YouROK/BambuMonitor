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
	s.Router.GET("/login", s.LoginGetHandler)
	s.Router.POST("/login", s.LoginPostHandler)
	s.Router.GET("/logout", s.LogoutHandler)

	protected := s.Router.Group("/")
	protected.Use(s.AuthMiddleware())
	{
		protected.GET("/", s.IndexHandler)
		protected.GET("/config", s.ConfigHandler)
		protected.GET("/status", s.PrinterStatus)
		protected.GET("/timelapse", s.TimelapsHandler)
		protected.GET("/tl/file/*path", s.TimelapsFile)
		protected.GET("/snap", s.SnapHandler)

		protected.POST("/config", s.ConfigSetter)
		protected.POST("/printer/light", s.ToggleLight)
		protected.POST("/printer/stop", s.StopPrinting)
		protected.POST("/printer/pause", s.TogglePause)
		protected.POST("/assemblevideo", s.HandleAssemble)
		protected.POST("/tl/remove", s.TimelapsRemove)
	}
}
