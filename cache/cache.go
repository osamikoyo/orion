package cache

import (
	"net/http"

	"github.com/osamikoyo/orion/config"
	"github.com/osamikoyo/orion/selfcach"
	"github.com/rs/zerolog"
)

type Cache struct {
	logger *zerolog.Logger
	cfg    *config.Config
	cache  *selfcach.Cache
}

func NewCache(sc *selfcach.Cache, logger *zerolog.Logger, cfg *config.Config) *Cache {
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
			w.Write([]byte(value))
			return
		}

		c.logger.Info().Msgf("not found url in cash, adding..")

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
