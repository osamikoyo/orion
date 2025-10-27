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
	Middleware func(next http.Handler) http.Handler

	Handler struct {
		proxy        *proxy.ProxyMW
		loadbalancer *loadbalancer.LoadBalancer
		cfg          *config.Config
		logger       *logger.Logger
		mws          map[string][]Middleware
	}
)

func NewHandler(proxy *proxy.ProxyMW, loadbalancer *loadbalancer.LoadBalancer, logger *logger.Logger, cfg *config.Config) *Handler {
	sc := selfcach.NewCache(logger, time.Hour, 3*time.Hour)
	cache := cache.NewCache(sc, logger, cfg)

	rate := rate.NewRateLimitingMiddleware(logger, cfg)

	auth := auth.NewAuthMW(cfg, logger)

	mws := make(map[string][]Middleware)

	for _, gateway := range cfg.Gateways {

		var mwArr []Middleware

		if gateway.Auth {
			mwArr = append(mwArr, auth.Middleware)
		}

		if gateway.Cash {
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
	now := time.Now()
	path := r.URL.Path

	defer func() {
		metrics.RequestTotal.WithLabelValues(path).Inc()
		metrics.RequestDuration.WithLabelValues(path).Observe(float64(time.Since(now).Seconds()))
	}()

	target, err := h.loadbalancer.Balance(r)
	if err != nil {
		h.logger.Error("failed balance",
			zap.String("path", r.URL.Path),
			zap.Error(err))

		http.Error(w, "failed balance targets", http.StatusBadGateway)

		metrics.ErrorRequestTotal.WithLabelValues(path).Inc()

		return
	}
	defer h.loadbalancer.DecTarget(target)

	prefix := "/" + strings.Split(r.URL.Path, "/")[1]

	var (
		mws []Middleware
		ok  bool
	)

	mws, ok = h.mws[prefix]
	if !ok {
		h.logger.Error("failed load mws",
			zap.String("prefix", prefix))

		mws = nil
	}

	proxymw := h.proxy.Middleware(target)

	for _, mw := range mws {
		proxymw = mw(proxymw).(http.HandlerFunc)
	}

	h.logger.Info("request was successfully setuped",
		zap.String("target", target),
		zap.String("prefix", prefix))

	proxymw.ServeHTTP(w, r)
}
