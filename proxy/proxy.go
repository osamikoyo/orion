package proxy

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/osamikoyo/orion/logger"
	"go.uber.org/zap"
)

type ProxyMW struct {
	logger *logger.Logger
}

func NewProxyMW(logger *logger.Logger) *ProxyMW {
	return &ProxyMW{
		logger: logger,
	}
}

func (mw *ProxyMW) Middleware(target string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		targetURL := strings.ReplaceAll(target, "{id}", chi.URLParam(r, "id"))

		mw.logger.Info("new api request", zap.String("target", target))

		proxy := httputil.NewSingleHostReverseProxy(&url.URL{Scheme: "http", Host: targetURL})
		proxy.ServeHTTP(w, r)
	}
}
