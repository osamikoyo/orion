package cache

import (
	"net/http"

	"github.com/osamikoyo/orion/config"
	"github.com/osamikoyo/orion/logger"
	"github.com/osamikoyo/orion/selfcach"
	"go.uber.org/zap"
)

type Cache struct {
	logger *logger.Logger
	cfg    *config.Config
	cache  *selfcach.Cache
}

func NewCache(sc *selfcach.Cache, logger *logger.Logger, cfg *config.Config) *Cache {
	return &Cache{
		logger: logger,
		cfg:    cfg,
	}
}

func (c *Cache) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.URL.Path

		value, ok := c.cache.Get(key)
		if ok {
			c.logger.Info("fetched cache for url", zap.String("url", key))

			w.Write([]byte(value))
			return
		}

		c.logger.Warn("not found url in cache", zap.String("url", key))

		wr := &responseWriter{ResponseWriter: w}
		next.ServeHTTP(wr, r)
		c.cache.Set(key, wr.body)
	})
}

type responseWriter struct {
	http.ResponseWriter
	body []byte
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	rw.body = append(rw.body, b...)
	return rw.ResponseWriter.Write(b)
}
