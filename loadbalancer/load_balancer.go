// load balance functionality
package loadbalancer

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strings"
	"sync"
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
	// urlInfo stores some information about targets
	urlInfo struct {
		health bool
		load   int
	}
	//LoadBalancer is main struct in package
	//route all requests and balance load between targets
	LoadBalancer struct {
		// healthchecker stores component for regularu health check
		healthchecker *healthchecker.HealthChecker
		// info stores map with urlInfo for every target
		info map[string]urlInfo
		// targets stores map with array of targets for every prefix
		targets map[string][]string
		// prefix stores map with prefix for every target
		prefix map[string]string

		// timerDur stores health check duration
		timerDur time.Duration

		logger *logger.Logger

		mu sync.RWMutex
	}
)

// NewLoadBalancer parse config in maps and create new LoadBalancer
func NewLoadBalancer(cfg *config.Config, logger *logger.Logger) (*LoadBalancer, error) {
	// Check configuration on valid
	if err := checkCfgValid(cfg); err != nil {
		logger.Error("config is not valid", zap.Error(err))
		return nil, err
	}

	logger.Info("setuping loadbalancer...")

	// Create healthchecker
	healthchecker := healthchecker.NewHealthChecker(cfg, logger)

	// Create maps
	targets := make(map[string][]string)
	prefix := make(map[string]string)
	info := make(map[string]urlInfo)

	// parse every gateway
	for _, gateway := range cfg.Gateways {
		targetsArr := make([]string, len(gateway.Targets))

		// iterate for targets
		for i, target := range gateway.Targets {
			prefix[target.Url] = gateway.Prefix
			targetsArr[i] = target.Url

			// add info for target url
			info[target.Url] = urlInfo{
				// by default all urls are healthy
				// and have zero load
				health: true,
				load:   0,
			}
		}

		// add targets in map
		targets[gateway.Prefix] = targetsArr
	}

	// setup load balancer
	lb := &LoadBalancer{
		healthchecker: healthchecker,
		targets:       targets,
		prefix:        prefix,
		info:          info,
		timerDur:      cfg.HealthCheckTimeout,
		logger:        logger,
	}

	// start regulary healthcheck
	lb.runHealthCheck(context.Background())

	return lb, nil
}

func (lb *LoadBalancer) Balance(r *http.Request) (string, error) {
	parts := strings.Split(r.URL.Path, "/")

	prefix := "/" + parts[1]

	mintarget, err := lb.selectTarget(prefix)
	if err != nil {
		lb.logger.Error("failed select minimal load target",
			zap.String("prefix", prefix),
			zap.Error(err))

		return "", ErrNotFound
	}

	lb.incLoad(mintarget)

	return mintarget, nil
}

func (lb *LoadBalancer) incLoad(target string) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	info, ok := lb.info[target]
	if !ok {
		lb.info[target] = urlInfo{
			health: false,
			load:   0,
		}

		return
	}

	lb.info[target] = urlInfo{
		health: info.health,
		load:   info.load + 1,
	}
}

func (lb *LoadBalancer) DecTarget(target string) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	info, ok := lb.info[target]
	if !ok {
		lb.info[target] = urlInfo{
			health: false,
			load:   0,
		}

		return
	}

	if info.load <= 0 {
		lb.info[target] = urlInfo{
			health: info.health,
			load:   0,
		}

		return
	}

	lb.info[target] = urlInfo{
		health: info.health,
		load:   info.load - 1,
	}
}

func (lb *LoadBalancer) runHealthCheck(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(lb.timerDur)

		for {
			select {
			case <-ticker.C:
				lb.logger.Info("starting healthcheck")

				health := lb.healthchecker.Check(ctx)

				lb.mu.Lock()

				for target, info := range lb.info {
					lb.info[target] = urlInfo{
						health: health[target],
						load:   info.load,
					}
				}

				lb.mu.Unlock()
			case <-ctx.Done():
				ticker.Stop()
				return
			}
		}

	}()
}

func (lb *LoadBalancer) selectTarget(prefix string) (string, error) {
	if len(prefix) == 0 {
		return "", errors.New("empty prefix")
	}

	lb.mu.RLock()

	targets, ok := lb.targets[prefix]
	if !ok {
		lb.logger.Error("failed to fetch targets",
			zap.String("prefix", prefix))

		lb.mu.RUnlock()

		return "", ErrNotFound
	}

	lb.mu.RUnlock()

	minLoadCount := MaxLoad
	minTarget := ""

	healthyTargets := 0

	for _, t := range targets {
		info, ok := lb.info[t]
		if !ok {
			lb.logger.Warn("failed to fetch info",
				zap.String("target", t))

			continue
		}

		if !info.health {
			lb.logger.Warn("unhealthy target",
				zap.String("url", t))

			continue
		}

		healthyTargets++

		if info.load == 0 {
			return t, nil
		}

		if info.load < minLoadCount {
			minLoadCount = info.load
			minTarget = t
		}
	}

	if healthyTargets == 0 {
		return "", ErrUnAvalaible
	}

	return minTarget, nil
}

func checkCfgValid(cfg *config.Config) error {
	doubles := make(map[string]int)

	if cfg == nil {
		return errors.New("empty config")
	}

	for _, gateway := range cfg.Gateways {
		if len(gateway.Targets) == 0 {
			return errors.New("empty targets")
		}

		if len(gateway.Prefix) == 0 {
			return errors.New("empty prefix")
		}

		doubles[gateway.Prefix]++
	}

	for prefix, count := range doubles {
		if count > 1 {
			return fmt.Errorf("double prefix: %s", prefix)
		}
	}

	return nil
}
