// load balance functionality
package loadbalancer

import (
	"context"
	"errors"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/osamikoyo/orion/config"
	"github.com/osamikoyo/orion/healthchecker"
	"github.com/osamikoyo/orion/logger"
	"go.uber.org/zap"
)

const MaxLoad = math.MaxInt32

var (
	ErrNotFound    = errors.New("not found service to route request")
	ErrUnAvalaible = errors.New("service is unavalaible")
)

type (
	Balancer interface {
		SelectTarget(prefix string) (string, error)
		SetHealthInfo(health map[string]bool)
	}

	LoadBalancer struct {
		balancer      Balancer
		logger        *logger.Logger
		healthchecker *healthchecker.HealthChecker
	}
)

func NewLoadBalancer(cfg *config.Config, logger *logger.Logger) (*LoadBalancer, error) {
	loadbalancer := &LoadBalancer{
		logger:        logger,
		healthchecker: healthchecker.NewHealthChecker(cfg, logger),
	}

	switch cfg.LoadBalancerAlg {
	case "wrr":
		loadbalancer.balancer = initWeightRoundRobin(cfg, logger)
	case "rr":
		loadbalancer.balancer = initRoundRobin(cfg, logger)
	default:
		logger.Error("unknown load balancer algorithm",
			zap.String("alg", cfg.LoadBalancerAlg))
	}

	health := make(chan map[string]bool, 1)

	go loadbalancer.runHealthCheck(context.Background(), health, cfg.HealthCheckTimeout)
	go func() {
		for {
			healthinfo := <-health
			loadbalancer.balancer.SetHealthInfo(healthinfo)
		}
	}()

	return loadbalancer, nil
}

func (lb *LoadBalancer) runHealthCheck(ctx context.Context, output chan map[string]bool, dur time.Duration) {
	ticker := time.NewTicker(dur)

	for {
		select {
		case <-ctx.Done():

			return

		case <-ticker.C:
			output <- lb.healthchecker.Check(ctx)
		}
	}
}

func (lb *LoadBalancer) Balance(r *http.Request) (string, error) {
	parts := strings.Split(r.URL.Path, "/")

	prefix := "/" + parts[1]

	return lb.balancer.SelectTarget(prefix)
}
