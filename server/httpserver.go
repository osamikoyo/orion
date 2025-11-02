package server

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/osamikoyo/orion/config"
	"github.com/osamikoyo/orion/logger"
	"go.uber.org/zap"
)

type HTTPServer struct {
	logger *logger.Logger
	srv    *http.Server
}

func NewHTTPServer(cfg *config.Config, logger *logger.Logger) *HTTPServer {
	return &HTTPServer{
		logger: logger,
		srv: &http.Server{
			Addr: cfg.Addr,
		},
	}
}

func (s *HTTPServer) Run() error {
	s.logger.Info("starting http server", zap.String("addr", s.srv.Addr))

	if err := s.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		s.logger.Error("failed start server", zap.Error(err))

		return err
	}

	return nil
}

func (s *HTTPServer) Shutdown(ctx context.Context) error {
	s.logger.Info("shutdown http server")

	if err := s.srv.Shutdown(ctx); err != nil {
		s.logger.Error("failed shutdown server", zap.Error(err))
		return err
	}

	return nil
}

func (s *HTTPServer) SetHandler(r chi.Router) {
	s.srv.Handler = r
}
