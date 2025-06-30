package main

import (
	"TemplatestPGSQL/internal/config"
	customLogger "TemplatestPGSQL/internal/logger"
	"TemplatestPGSQL/internal/repo"
	"TemplatestPGSQL/internal/service"
	"context"
	"github.com/joho/godotenv"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/kelseyhightower/envconfig"
)

func main() {

	// Config
	if err := godotenv.Load(config.EnvPath); err != nil {
		log.Fatalf("failed to load config file %s: %v", config.EnvPath, err)
	}

	var cfg config.AppConfig
	if err := envconfig.Process("", &cfg); err != nil {
		log.Fatalf("failed to process config file %s: %v", config.EnvPath, err)
	}

	// Logger
	logger, err := customLogger.NewLogger(cfg.LogLevel)
	if err != nil {
		log.Fatal("failed to initialize logger: ", err)
	}

	// Repository
	repository, err := repo.NewRepository(context.Background(), cfg.Memory)
	if err != nil {
		log.Fatal("failed to initialize repo: ", err)
	}
	err = repository.InitTables(context.Background())
	if err != nil {
		log.Fatal("failed to set up repo: ", err)
	}

	// Service initialization
	serviceInstance := service.NewService(repository, logger, cfg.GRPC)

	// Listening and serving
	go func() {
		logger.Infof("Starting server on %s", cfg.GRPC.Port)
		if err := serviceInstance.ListenAndServe(); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	// Fold operations
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)
	<-signalChan

	logger.Info("Shutting down gracefully...")
}
