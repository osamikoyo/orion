// rate limiting middleware
package rate

import (
	"net/http"
	"time"

	"github.com/osamikoyo/orion/config"
	"github.com/osamikoyo/orion/logger"
	"github.com/osamikoyo/orion/metrics"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

type RateLimitingMW struct {
	cfg    *config.Config
	logger *logger.Logger
}

func NewRateLimitingMiddleware(logger *logger.Logger, cfg *config.Config) *RateLimitingMW {
	return &RateLimitingMW{
		cfg:    cfg,
		logger: logger,
	}
}

func (mw *RateLimitingMW) RateLimitMiddleware(next http.Handler) http.Handler {
	limiter := rate.NewLimiter(rate.Every(time.Minute/100), mw.cfg.RateLimiting.MaxRequest)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !limiter.Allow() {
			mw.logger.Warn("rate limiter lock",
				zap.String("remote_addr", r.RemoteAddr))

			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)

			metrics.ErrorRequestTotal.WithLabelValues(r.URL.Path).Inc()

			return
		}
		next.ServeHTTP(w, r)
	})
}
