package loadbalancer

import (
	"sync"

	"github.com/osamikoyo/orion/config"
	"github.com/osamikoyo/orion/errors"
	"github.com/osamikoyo/orion/logger"
	"go.uber.org/zap"
)

type (
	wrrexpTarget struct {
		url    string
		health bool
		weight int
	}

	wrrexpandedTargets struct {
		targets     []wrrexpTarget
		current     int
		totalWeight int
	}

	WeightRoundRobinBalancer struct {
		logger      *logger.Logger
		targetsInfo map[string]*wrrexpandedTargets
		mu          sync.RWMutex
	}
)

func initWeightRoundRobin(cfg *config.Config, logger *logger.Logger) *WeightRoundRobinBalancer {
	targetsInfo := make(map[string]*wrrexpandedTargets)

	for _, gateway := range cfg.Gateways {
		targets := make([]wrrexpTarget, 0, len(gateway.Targets))
		totalWeight := 0

		for _, target := range gateway.Targets {
			if target.Weight <= 0 {
				logger.Warn("target weight is zero or negative, setting to 1", zap.String("url", target.Url))
				target.Weight = 1
			}
			targets = append(targets, wrrexpTarget{
				url:    target.Url,
				health: true,
				weight: target.Weight,
			})
			totalWeight += target.Weight
		}

		targetsInfo[gateway.Prefix] = &wrrexpandedTargets{
			targets:     targets,
			current:     0,
			totalWeight: totalWeight,
		}
	}

	return &WeightRoundRobinBalancer{
		logger:      logger,
		targetsInfo: targetsInfo,
	}
}

func (wrrb *WeightRoundRobinBalancer) SetHealthInfo(healthy map[string]bool) {
	wrrb.mu.Lock()
	defer wrrb.mu.Unlock()

	for prefix, exp := range wrrb.targetsInfo {
		newTotalWeight := 0
		changed := false

		for i := range exp.targets {
			oldHealth := exp.targets[i].health
			url := exp.targets[i].url

			isHealth, ok := healthy[url]
			if !ok {
				isHealth = true
			}

			if oldHealth != isHealth {
				changed = true
			}

			exp.targets[i].health = isHealth
			if isHealth {
				newTotalWeight += exp.targets[i].weight
			}
		}

		if changed {
			exp.totalWeight = newTotalWeight
			exp.current = 0
			wrrb.logger.Info("health status updated, new total weight",
				zap.String("prefix", prefix),
				zap.Int("total_weight", newTotalWeight))
		}
	}
}

func (wrrb *WeightRoundRobinBalancer) SelectTarget(prefix string) (string, error) {
	wrrb.mu.RLock()
	exp, ok := wrrb.targetsInfo[prefix]
	wrrb.mu.RUnlock()
	if !ok {
		wrrb.logger.Error("could not found targets for prefix",
			zap.String("prefix", prefix))
		return "", errors.ErrPrefixNotFound
	}

	if exp.totalWeight == 0 {
		return "", errors.ErrNoHealthyTargets
	}

	wrrb.mu.Lock()
	defer wrrb.mu.Unlock()

	for attempts := 0; attempts < len(exp.targets); attempts++ {
		currentTarget := &exp.targets[exp.current]
		exp.current = (exp.current + 1) % len(exp.targets)

		if !currentTarget.health {
			continue
		}

		if currentTarget.weight > 0 {
			return currentTarget.url, nil
		}
	}

	for _, t := range exp.targets {
		if t.health {
			return t.url, nil
		}
	}

	return "", errors.ErrNoHealthyTargets
}
