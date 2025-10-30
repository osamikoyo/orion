package server

import (
	"fmt"
	"log"

	"github.com/corazawaf/coraza/v3"
	"github.com/corazawaf/coraza/v3/types"
	"github.com/osamikoyo/orion/config"
	"github.com/osamikoyo/orion/logger"
	"go.uber.org/zap"
)

func logError(error types.MatchedRule) {
	msg := error.ErrorLog()
	log.Printf("[logError][%s] %s\n", error.Rule().Severity(), msg)
}

func newWaf(cfg *config.Config, logger *logger.Logger) (coraza.WAF, error) {
	waf, err := coraza.NewWAF(coraza.NewWAFConfig().
		WithDirectivesFromFile(cfg.WAF.ConfigPath).
		WithErrorCallback(logError))
	if err != nil {
		logger.Error("failed create waf",
			zap.String("config_path", cfg.WAF.ConfigPath),
			zap.Error(err))

		return nil, fmt.Errorf("failed create waf: %v", err)
	}

	logger.Info("waf was created successfully")

	return waf, nil
}
