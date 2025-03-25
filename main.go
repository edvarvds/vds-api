package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"api_vds/internal/api"
	"api_vds/internal/config"
	"api_vds/internal/cache"
)

func main() {
	// Carregar configuração
	cfg, err := config.LoadConfig("config/config.yaml")
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	// Inicializar cliente Redis
	redisClient, err := cache.NewRedisClient(cfg)
	if err != nil {
		log.Fatalf("Error connecting to Redis: %v", err)
	}

	// Criar e iniciar servidor
	server := api.NewServer(cfg, redisClient)
	log.Printf("Server starting on port %s", cfg.Server.Port)
	if err := server.Start(); err != nil {
		log.Fatalf("Error starting server: %v", err)
	}

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	// Graceful shutdown
	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exiting")
} 