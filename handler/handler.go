package handler

import (
	"net/http"

	"github.com/osamikoyo/orion/config"
	loadbalancer "github.com/osamikoyo/orion/laodbalancer"
	"github.com/osamikoyo/orion/proxy"
)

type Handler struct {
	proxy        *proxy.ProxyMW
	loadbalancer *loadbalancer.LoadBalancer
	cfg          *config.Config
}

func NewHandler(proxy *proxy.ProxyMW, loadbalancer *loadbalancer.LoadBalancer, cfg *config.Config) *Handler {
	return &Handler{
		proxy:        proxy,
		loadbalancer: loadbalancer,
		cfg:          cfg,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	target, err := h.loadbalancer.Balance(r)
	if err != nil {
		http.Error(w, "failed balance targets", http.StatusBadGateway)

		return
	}
	defer h.loadbalancer.DecTarget(target)

	mw := h.proxy.Middleware(target)

	mw.ServeHTTP(w, r)
}
