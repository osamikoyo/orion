package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/osamikoyo/orion/config"
	"github.com/osamikoyo/orion/logger"
	"github.com/quic-go/quic-go/http3"
	"go.uber.org/zap"
)

type HTTP3Server struct {
	logger *logger.Logger
	srv    *http3.Server
}

func loadTLSConfig(certFile, keyFile string) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, err
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		CipherSuites: []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		},
		PreferServerCipherSuites: true,
	}, nil
}

func NewHTTP3Server(cfg *config.Config, logger *logger.Logger) (*HTTP3Server, error) {
	logger.Info("setup tls...",
		zap.String("cert", cfg.TLS.Cert),
		zap.String("key", cfg.TLS.Key))

	tlsconfig, err := loadTLSConfig(cfg.TLS.Cert, cfg.TLS.Key)
	if err != nil {
		logger.Error("failed to load tls config",
			zap.String("cert", cfg.TLS.Cert),
			zap.String("key", cfg.TLS.Key),
			zap.Error(err))

		return nil, fmt.Errorf("failed load tls config: %v", err)
	}

	logger.Info("tls config was successfully setuped")

	return &HTTP3Server{
		logger: logger,
		srv: &http3.Server{
			Addr:      cfg.Addr,
			TLSConfig: tlsconfig,
		},
	}, nil
}

func (h *HTTP3Server) Run() error {
	h.logger.Info("starting http3 server", zap.String("addr", h.srv.Addr))

	if err := h.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		h.logger.Error("failed to start http3 server",
			zap.Error(err))

		return err
	}

	return nil
}

func (h *HTTP3Server) Shutdown(ctx context.Context) error {
	h.logger.Info("shutdowning http3 server")

	if err := h.srv.Shutdown(ctx); err != nil {
		h.logger.Error("failed to shutdown http3 server",
			zap.Error(err))

		return err
	}

	return nil
}

func (h *HTTP3Server) SetHandler(r chi.Router) {
	h.srv.Handler = r
}
