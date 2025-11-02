package server

import (
	"context"
	"fmt"
	"net/http"

	txhttp "github.com/corazawaf/coraza/v3/http"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/osamikoyo/orion/config"
	"github.com/osamikoyo/orion/handler"
	"github.com/osamikoyo/orion/loadbalancer"
	"github.com/osamikoyo/orion/logger"
	"github.com/osamikoyo/orion/proxy"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

const (
	HttpProto  = "http"
	Http3Proto = "http3"
)

type (
	Server interface {
		Run() error
		SetHandler(r chi.Router)
		Shutdown(context.Context) error
	}
)

func NewServer(r *chi.Mux, logger *logger.Logger, cfg *config.Config) (Server, context.CancelFunc, error) {
	loadbalancer, cancel, err := loadbalancer.NewLoadBalancer(cfg, logger)
	if err != nil {
		logger.Error("failed create load balancer", zap.Error(err))

		return nil, nil, err
	}

	proxy := proxy.NewProxyMW(logger)

	handler := handler.NewHandler(proxy, loadbalancer, logger, cfg)

	if err := setupRouters(r, handler, cfg, logger); err != nil {
		logger.Error("failed setup routers", zap.Error(err))

		return nil, nil, err
	}

	server, err := setupServer(cfg, logger)
	if err != nil {
		logger.Error("failed to setup server",
			zap.Error(err))

		return nil, nil, err
	}

	server.SetHandler(r)

	return server, cancel, nil
}

func setupRouters(r *chi.Mux, handler *handler.Handler, cfg *config.Config, logger *logger.Logger) error {
	if cfg.CORS.Use {
		r.Use(cors.Handler(cors.Options{
			AllowedMethods: cfg.CORS.AllowMethods,
			AllowedHeaders: cfg.CORS.AllowHeaders,
			AllowedOrigins: cfg.CORS.AllowOrigins,
			MaxAge:         cfg.CORS.MaxAge,
		}))
	}

	r.Use(middleware.Logger, middleware.Recoverer, middleware.Timeout(cfg.RequestTimeout))

	r.Get("/hell", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "heaven")
	})

	r.Handle("/metrics", promhttp.Handler())

	if cfg.WAF.Use {
		waf, err := newWaf(cfg, logger)
		if err != nil {
			logger.Error("failed create new waf",
				zap.Error(err))

			return err
		}
		r.Handle("/*", txhttp.WrapHandler(waf, handler))

	} else {
		r.HandleFunc("/*", handler.ServeHTTP)
	}
	return nil
}

func setupServer(cfg *config.Config, logger *logger.Logger) (Server, error) {
	if cfg.Proto == HttpProto {
		return NewHTTPServer(cfg, logger), nil
	}

	if cfg.Proto == Http3Proto {
		return NewHTTP3Server(cfg, logger)
	}

	logger.Error("unknown proto type", zap.String("proto", cfg.Proto))

	return nil, fmt.Errorf("unknown proto type: %s", cfg.Proto)
}
