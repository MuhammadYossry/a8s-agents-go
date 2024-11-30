package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize components
	broker := NewPubSub()
	executor := NewTaskExecutor()
	metrics := NewMetrics()
	registry := NewCapabilityRegistry()

	agent := NewAgent(broker, executor, metrics, registry)

	// Start agent
	if err := agent.Start(ctx); err != nil {
		log.Fatalf("Failed to start agent: %v", err)
	}

	// Wait for shutdown signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := agent.Stop(shutdownCtx); err != nil {
		log.Printf("Error during shutdown: %v", err)
	}
}
