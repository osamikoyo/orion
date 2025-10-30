// main handler
package handler

import (
	"net/http"
	"strings"
	"time"

	"github.com/osamikoyo/orion/auth"
	"github.com/osamikoyo/orion/cache"
	"github.com/osamikoyo/orion/config"
	"github.com/osamikoyo/orion/loadbalancer"
	"github.com/osamikoyo/orion/logger"
	"github.com/osamikoyo/orion/metrics"
	"github.com/osamikoyo/orion/proxy"
	"github.com/osamikoyo/orion/rate"
	"github.com/osamikoyo/orion/selfcach"
	"go.uber.org/zap"
)

type (
	// middleware type
	Middleware func(next http.Handler) http.Handler

	// Handler struct
	Handler struct {
		// proxy stores proxy middleware
		proxy        *proxy.ProxyMW
		loadbalancer *loadbalancer.LoadBalancer
		cfg          *config.Config
		logger       *logger.Logger
		// mws stores middlewares for each prefix
		mws map[string][]Middleware
	}
)

// cunstructor for Handler
func NewHandler(proxy *proxy.ProxyMW, loadbalancer *loadbalancer.LoadBalancer, logger *logger.Logger, cfg *config.Config) *Handler {
	// create selfcache and cache middleware
	sc := selfcach.NewCache(logger, time.Hour, 3*time.Hour)
	cache := cache.NewCache(sc, logger, cfg)

	// create rate middleware
	rate := rate.NewRateLimitingMiddleware(logger, cfg)

	// create auth middleware
	auth := auth.NewAuthMW(cfg, logger)

	// create mws map
	mws := make(map[string][]Middleware)

	for _, gateway := range cfg.Gateways {
		// itarate every gateway and its middlewares

		var mwArr []Middleware

		// append middlewares, witch were in config
		if gateway.Auth {
			mwArr = append(mwArr, auth.Middleware)
		}

		if gateway.Cache {
			mwArr = append(mwArr, cache.Middleware)
		}

		if gateway.Rate {
			mwArr = append(mwArr, rate.RateLimitMiddleware)
		}

		mws[gateway.Prefix] = mwArr
	}

	return &Handler{
		proxy:        proxy,
		loadbalancer: loadbalancer,
		cfg:          cfg,
		mws:          mws,
		logger:       logger,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// store values for metrics
	now := time.Now()
	path := r.URL.Path

	// set metrics
	defer func() {
		metrics.RequestTotal.WithLabelValues(path).Inc()
		metrics.RequestDuration.WithLabelValues(path).Observe(float64(time.Since(now).Seconds()))
	}()

	// get target
	target, err := h.loadbalancer.Balance(r)
	if err != nil {
		h.logger.Error("failed balance",
			zap.String("path", r.URL.Path),
			zap.Error(err))

		http.Error(w, "failed balance targets", http.StatusBadGateway)

		return
	}

	prefix := "/" + strings.Split(r.URL.Path, "/")[1]

	var (
		mws []Middleware
		ok  bool
	)

	// get mws by prefix
	mws, ok = h.mws[prefix]
	if !ok {
		h.logger.Error("failed load mws",
			zap.String("prefix", prefix))

		mws = nil
	}

	// get proxy handler
	proxymw := h.proxy.Middleware(target)

	for _, mw := range mws {
		//set proxy wm with every mws
		proxymw = mw(proxymw).(http.HandlerFunc)
	}

	h.logger.Info("request was successfully setuped",
		zap.String("target", target),
		zap.String("prefix", prefix))

	proxymw.ServeHTTP(w, r)
}
