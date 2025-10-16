package server

import (
	"context"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/osamikoyo/orion/auth"
	"github.com/osamikoyo/orion/cache"
	"github.com/osamikoyo/orion/config"
	"github.com/osamikoyo/orion/proxy"
	"github.com/osamikoyo/orion/rate"
	"github.com/osamikoyo/orion/selfcach"
	"github.com/quic-go/quic-go/http3"
	"github.com/rs/zerolog"
)

type Server struct {
	router chi.Router
	logger *zerolog.Logger
	cfg    *config.Config
	h3S    *http3.Server
}

func NewServer(r chi.Router, logger *zerolog.Logger, cfg *config.Config) *Server {
	cache := cache.NewCache(
		selfcach.NewCache(10*time.Minute, 5*time.Hour),
		logger,
		cfg,
	)

	auth := auth.NewAuthMW(&cfg.AuthConfig, logger)
	rl := rate.NewRateLimitingMiddleware(logger, &cfg.RateLimitingConfig)
	proxy := proxy.NewProxyMW(logger)

	r.Route("/api", func(r chi.Router) {
		for _, gateway := range cfg.Gateways {
			route := r

			if gateway.Auth {
				route = route.With(auth.Middleware)
			}

			if gateway.Cash {
				route = route.With(cache.Middleware)
			}

			route.HandleFunc(gateway.Prefix+"/*", proxy.Middleware(gateway.Target))
		}
	})

	if cfg.RateLimitingConfig.Use {
		r.Use(rl.RateLimitMiddleware)
	}

	server := &http3.Server{
		Addr:    cfg.Addr,
		Handler: r,
	}

	return &Server{
		router: r,
		logger: logger,
		cfg:    cfg,
		h3S:    server,
	}
}

func (s *Server) Run() error {
	if err := s.h3S.ListenAndServe(); err != nil {
		s.logger.Error().Msgf("failed listen and serve on %s: %v", s.cfg.Addr, err)

		return err
	}

	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	if err := s.h3S.Shutdown(ctx); err != nil {
		s.logger.Error().Msgf("failed shutdown server: %v", err)
		return err
	}

	return nil
}
