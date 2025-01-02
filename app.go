package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/Relax-N-Tax/AgentNexus/hub"
	"github.com/Relax-N-Tax/AgentNexus/orchestrator"
	"github.com/Relax-N-Tax/AgentNexus/types"
)

type Application struct {
	hub          *hub.Server
	orchestrator *orchestrator.Orchestrator
	config       *Config
}

func NewApplication(config *Config) (*Application, error) {
	// Create hub server
	hubServer, err := hub.NewServer(hub.DefaultConfig())
	if err != nil {
		return nil, fmt.Errorf("failed to initialize HubServer: %w", err)
	}

	internalConfig := types.InternalAgentConfig{
		LLMConfig: struct {
			BaseURL string
			APIKey  string
			Model   string
			Timeout time.Duration
		}{
			BaseURL: config.LLM.BaseURL,
			APIKey:  config.LLM.APIKey,
			Model:   config.LLM.Model,
			Timeout: config.LLM.Timeout,
		},
	}

	// Create orchestrator with New instead of NewWithRegistry
	orch, err := orchestrator.New(orchestrator.Config{
		Agents:         config.Agents,
		InternalConfig: internalConfig,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create orchestrator: %w", err)
	}

	return &Application{
		hub:          hubServer,
		orchestrator: orch,
		config:       config,
	}, nil
}

func (app *Application) processUserInput(ctx context.Context) error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Printf("%sEnter your task (press Enter to submit): %s", boldTeal, reset)
	userTaskMsg, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read user input: %w", err)
	}

	userTaskMsg = strings.TrimSpace(userTaskMsg)
	if userTaskMsg == "" {
		return fmt.Errorf("task message cannot be empty")
	}

	ctx, err = app.orchestrator.ProcessQuery(ctx, userTaskMsg)
	if err != nil {
		return fmt.Errorf("failed to process query: %w", err)
	}

	if result, ok := ctx.Value(types.TaskExtractionResultKey).(string); ok {
		log.Printf("%sExtracted Task:%s\n%s\n", teal, reset, result)
	}

	return nil
}

func (app *Application) Shutdown(ctx context.Context) error {
	doneChan := make(chan struct{})

	go func() {
		var wg sync.WaitGroup
		wg.Add(2)

		// Shutdown hub
		go func() {
			defer wg.Done()
			if err := app.hub.Shutdown(ctx); err != nil {
				log.Printf("Hub shutdown error: %v", err)
			}
		}()

		// Shutdown orchestrator
		go func() {
			defer wg.Done()
			if err := app.orchestrator.Shutdown(ctx); err != nil {
				log.Printf("Orchestrator shutdown error: %v", err)
			}
		}()

		wg.Wait()
		close(doneChan)
	}()

	select {
	case <-doneChan:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (app *Application) gracefulShutdown(ctx context.Context) {
	log.Println("Initiating shutdown...")

	shutdownCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	if err := app.Shutdown(shutdownCtx); err != nil {
		log.Printf("Shutdown error: %v", err)
	}

	log.Println("Shutdown complete")
	os.Exit(0)
}
