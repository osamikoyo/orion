// cache middleware
package cache

import (
	"net/http"

	"github.com/osamikoyo/orion/config"
	"github.com/osamikoyo/orion/logger"
	"github.com/osamikoyo/orion/selfcach"
	"go.uber.org/zap"
)

// Cache stores components for middleware
type Cache struct {
	logger *logger.Logger
	cfg    *config.Config
	cache  *selfcach.Cache
}

// NewCache() creates new Cache
func NewCache(sc *selfcach.Cache, logger *logger.Logger, cfg *config.Config) *Cache {
	return &Cache{
		logger: logger,
		cfg:    cfg,
	}
}

// Middleware() creates cache middleware
func (c *Cache) Middleware(next http.Handler) http.Handler {
	// return handler
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.URL.Path

		// try to get cache for url
		value, ok := c.cache.Get(key)
		if ok {
			// if it was successfully found
			// write response from cache
			c.logger.Info("fetched cache for url", zap.String("url", key))

			w.Write([]byte(value))
			return
		}

		c.logger.Warn("not found url in cache", zap.String("url", key))
		// if not
		// create custome response writer and save response body in cache

		wr := &responseWriter{ResponseWriter: w}
		next.ServeHTTP(wr, r)
		c.cache.Set(key, wr.body)
	})
}

// custom response writer
type responseWriter struct {
	http.ResponseWriter
	body []byte
}

// io.Writer realization
func (rw *responseWriter) Write(b []byte) (int, error) {
	rw.body = append(rw.body, b...)
	return rw.ResponseWriter.Write(b)
}
