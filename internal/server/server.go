package server

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"go02-neighbor-help/internal/config"
	"go02-neighbor-help/internal/middleware"
)

type Server struct {
	httpServer *http.Server
	cfg        config.Config
}

func New(cfg config.Config, handler http.Handler) *Server {
	wrapped := middleware.Chain(
		middleware.Recover(),
		middleware.CORS(cfg.CORSOrigins),
		middleware.Logger(log.Printf),
	)(handler)
	return &Server{
		cfg: cfg,
		httpServer: &http.Server{
			Addr:         cfg.Addr,
			Handler:      wrapped,
			ReadTimeout:  cfg.ReadTimeout,
			WriteTimeout: cfg.WriteTimeout,
			IdleTimeout:  60 * time.Second,
		},
	}
}

func (s *Server) Start() error {
	log.Printf("server: listening on %s", s.cfg.Addr)
	err := s.httpServer.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	log.Println("server: shutting down...")
	return s.httpServer.Shutdown(ctx)
}

func (s *Server) HTTPServer() *http.Server { return s.httpServer }

func (s *Server) Addr() string { return s.cfg.Addr }
