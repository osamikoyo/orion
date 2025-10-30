// load balance functionality
package loadbalancer

import (
	"context"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/osamikoyo/orion/config"
	"github.com/osamikoyo/orion/errors"
	"github.com/osamikoyo/orion/healthchecker"
	"github.com/osamikoyo/orion/logger"
	"go.uber.org/zap"
)

const MaxLoad = math.MaxInt32

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

func NewLoadBalancer(cfg *config.Config, logger *logger.Logger) (*LoadBalancer, context.CancelFunc, error) {
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

		return nil, nil, errors.ErrNoHealthyTargets
	}

	health := make(chan map[string]bool, 1)

	ctx, cancel := context.WithCancel(context.Background())

	go loadbalancer.runHealthCheck(ctx, health, cfg.HealthCheckTimeout)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case healthinfo := <-health:
				loadbalancer.balancer.SetHealthInfo(healthinfo)
			}
		}
	}()

	return loadbalancer, cancel, nil
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
