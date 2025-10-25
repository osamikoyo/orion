// package with health check functionality
package healthchecker

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/osamikoyo/orion/config"
	"github.com/osamikoyo/orion/logger"
	"go.uber.org/zap"
)

// health checker stores components to create health check
type HealthChecker struct {
	// healthEndPoints stores endpoint for health request for every target
	healthEndPoints map[string]string
	logger          *logger.Logger
}

// NewHealthChecker create HealtchChecker and parse targets in map
func NewHealthChecker(cfg *config.Config, logger *logger.Logger) *HealthChecker {
	// create healthEndPoints map
	health := make(map[string]string)

	// parse every gateway
	for _, gateway := range cfg.Gateways {
		for _, target := range gateway.Targets {
			health[target.Url] = target.HealthEndpoint
		}
	}

	return &HealthChecker{
		healthEndPoints: health,
		logger:          logger,
	}
}

// Check starts health check for every target in config
// and give map with targets and their health
func (hc *HealthChecker) Check(_ context.Context) map[string]bool {
	// create result map
	health := make(map[string]bool)

	// set sync variables
	var (
		wg sync.WaitGroup
		mu sync.Mutex
	)

	// send http request on every health endpoint
	for target, endpoint := range hc.healthEndPoints {
		// start request gourutine every iteration
		wg.Go(func() {
			path := fmt.Sprintf("http://%s%s", target, endpoint)

			hc.logger.Info("forming healthcheck request",
				zap.String("target", target))
			// create http request with context
			_, err := http.Get(path)
			if err != nil {
				// if unhealthy
				hc.logger.Warn("unhealthy service", zap.String("url", path))

				mu.Lock()

				health[target] = false

				mu.Unlock()
			} else {
				// if healthy
				mu.Lock()

				health[target] = true

				mu.Unlock()
			}
		})
	}

	wg.Wait()

	return health
}
