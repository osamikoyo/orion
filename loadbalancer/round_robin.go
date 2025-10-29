package loadbalancer

import (
	"errors"
	"sync"

	"github.com/osamikoyo/orion/config"
	"github.com/osamikoyo/orion/logger"
	"go.uber.org/zap"
)

var (
	ErrPrefixNotFound = errors.New("not found prefix for url")
)

type (
	rrexpTarget struct {
		url    string
		health bool
	}

	rrexpandedTargets struct {
		targets []rrexpTarget
		index   int
	}

	RoundRobinBalancer struct {
		logger      *logger.Logger
		targetsInfo map[string]rrexpandedTargets
		mu          sync.RWMutex
	}
)

func initRoundRobin(cfg *config.Config, logger *logger.Logger) *RoundRobinBalancer {
	targetsInfo := make(map[string]rrexpandedTargets)

	for _, gateway := range cfg.Gateways {
		targets := make([]rrexpTarget, len(gateway.Targets))

		for i, target := range gateway.Targets {
			targets[i] = rrexpTarget{
				url:    target.Url,
				health: true,
			}
		}

		targetsInfo[gateway.Prefix] = rrexpandedTargets{
			targets: targets,
			index:   0,
		}
	}

	return &RoundRobinBalancer{
		logger:      logger,
		targetsInfo: targetsInfo,
	}
}

func (rrb *RoundRobinBalancer) SetHealthInfo(healthy map[string]bool) {
	rrb.mu.Lock()
	defer rrb.mu.Unlock()

	for prefix, exptarget := range rrb.targetsInfo {
		changed := false
		for i := range exptarget.targets {
			oldHealth := exptarget.targets[i].health
			url := exptarget.targets[i].url

			isHealth, ok := healthy[url]
			if !ok {
				isHealth = true
			}

			if oldHealth != isHealth {
				changed = true
			}

			exptarget.targets[i].health = isHealth
		}

		if changed {
			rrb.logger.Info("health status updated",
				zap.String("prefix", prefix))
		}
	}
}

func (rrb *RoundRobinBalancer) SelectTarget(prefix string) (string, error) {
	rrb.mu.RLock()

	expTargets, ok := rrb.targetsInfo[prefix]

	rrb.mu.RUnlock()
	if !ok {

		rrb.logger.Error("could not found targets",
			zap.String("prefix", prefix))

		return "", ErrPrefixNotFound
	}

	rrb.mu.Lock()

	target := expTargets.targets[expTargets.index]

	if expTargets.index == len(expTargets.targets)-1 {
		expTargets.index = 0
	} else {
		expTargets.index++
	}

	rrb.mu.Unlock()

	return target.url, nil
}
