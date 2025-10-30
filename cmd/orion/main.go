package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"

	"github.com/go-chi/chi/v5"
	"github.com/osamikoyo/orion/config"
	"github.com/osamikoyo/orion/logger"
	"github.com/osamikoyo/orion/server"
	"go.uber.org/zap"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	logger.Init(logger.Config{
		LogFile:   "orion.log",
		LogLevel:  "info",
		AddCaller: false,
		AppName:   "orion",
	})

	logger := logger.Get()

	logger.Info("starting setup orion")

	cfgpath := "config.yaml"

	if len(os.Args) > 1 {
		for i, arg := range os.Args {
			if arg == "--config" {
				cfgpath = os.Args[i+1]
			}
		}
	}

	cfg, err := config.NewConfig(cfgpath)
	if err != nil {
		logger.Error("failed to load config",
			zap.String("path", cfgpath),
			zap.Error(err))
	}

	logger.Info("setup router...")

	r := chi.NewRouter()

	server, cancelLoad, err := server.NewServer(r, logger, cfg)
	if err != nil {
		logger.Error("failed to setup server", zap.Error(err))

		return
	}

	logger.Info("starting orion",
		zap.Any("config", cfg))

	go func() {
		if err = server.Run(); err != nil && err != http.ErrServerClosed {
			logger.Error("failed to start orion", zap.Error(err))
		}
	}()

	logger.Info("orion successfully started!")

	<-ctx.Done()

	if err = server.Shutdown(ctx); err != nil {
		logger.Error("failed to shutdown orion", zap.Error(err))
	}

	cancelLoad()
}
