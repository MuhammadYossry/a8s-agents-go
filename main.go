package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/Relax-N-Tax/AgentNexus/hub"
	"github.com/Relax-N-Tax/AgentNexus/orchestrator"
	"github.com/Relax-N-Tax/AgentNexus/types"
	"gopkg.in/yaml.v3"
)

type Application struct {
	hub          *hub.Server
	orchestrator *orchestrator.Orchestrator
	config       *Config
}

type LLMConfig struct {
	BaseURL string        `yaml:"base_url"`
	APIKey  string        `yaml:"api_key"`
	Model   string        `yaml:"model"`
	Timeout time.Duration `yaml:"timeout"`
}

type Config struct {
	LLM    LLMConfig `yaml:"llm"`
	Agents []string  `yaml:"agents"`
}

func loadConfigFile(filename string) ([]byte, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var filteredLines []string
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && !strings.HasPrefix(trimmed, "#") {
			filteredLines = append(filteredLines, line)
		}
	}

	return []byte(strings.Join(filteredLines, "\n")), nil
}

func NewConfig() (*Config, error) {
	// Default configuration
	config := &Config{
		LLM: LLMConfig{
			Model:   "Qwen-2.5-72B-Chat",
			Timeout: 50 * time.Second,
		},
	}

	// Try to read from a8s.conf first
	data, err := loadConfigFile("a8s.conf")
	if err == nil {
		log.Printf("==> loading configuration from a8s.conf...")
		if err := yaml.Unmarshal(data, config); err != nil {
			return nil, fmt.Errorf("failed to parse a8s.conf: %w", err)
		}

		// Debug: Print loaded configuration
		log.Printf("==> Config loaded with:\n BaseURL: %s\n Model: %s", config.LLM.BaseURL, config.LLM.Model)
	} else {
		log.Printf("WARNING: No a8s.conf is missing..\n Using environment variables")
		// Only use environment variables if config file is not found
		config.LLM.BaseURL = os.Getenv("RNT_OPENAI_URL")
		config.LLM.APIKey = os.Getenv("RNT_OPENAI_API_KEY")
	}

	// Validate required fields
	if config.LLM.BaseURL == "" {
		return nil, errors.New("LLM base_url must be specified in config file or RNT_OPENAI_URL environment variable")
	}
	if config.LLM.APIKey == "" {
		return nil, errors.New("LLM api_key must be specified in config file or RNT_OPENAI_API_KEY environment variable")
	}

	return config, nil
}

func NewApplication(config *Config) (*Application, error) {
	// Create hub server with registry
	hubRegistry, err := hub.NewSQLiteRegistry()
	if err != nil {
		return nil, err
	}
	hubServer := hub.NewServer(hub.DefaultConfig(), hubRegistry)

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

func createInitialTask() *types.Task {
	return &types.Task{
		ID:          "task1",
		Title:       "Create a REST API endpoint",
		Description: "Create a REST API endpoint using Python",
		Requirements: types.TaskRequirement{
			SkillPath: types.TaskPath{"Development", "Backend", "Python", "CodeGeneration"},
			Action:    "generateCode",
			Parameters: map[string]interface{}{
				"framework": "FastAPI",
			},
		},
		Status:    types.TaskStatusPending,
		CreatedAt: time.Now(),
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

func main() {
	displayBanner()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	config, err := NewConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	app, err := NewApplication(config)
	if err != nil {
		log.Fatalf("Failed to create application: %v", err)
	}

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		app.gracefulShutdown(context.Background())
	}()

	ctx, err = app.orchestrator.Start(ctx)
	if err != nil {
		log.Fatalf("Failed to start application: %v", err)
	}
	err = app.hub.Start(ctx)
	if err != nil {
		log.Fatalf("Failed to start a8s hub: %v", err)
	}

	if err := app.processUserInput(ctx); err != nil {
		log.Printf("Error processing user input: %v", err)
	}

	<-ctx.Done()
}

const (
	reset    = "\033[0m"
	teal     = "\033[36m"
	boldTeal = "\033[1;36m"
)

func displayBanner() {
	banner := `
    █████╗  ██████╗ ███████╗███╗   ██╗████████╗███╗   ██╗███████╗██╗  ██╗██╗   ██╗███████╗
   ██╔══██╗██╔════╝ ██╔════╝████╗  ██║╚══██╔══╝████╗  ██║██╔════╝╚██╗██╔╝██║   ██║██╔════╝
   ███████║██║  ███╗█████╗  ██╔██╗ ██║   ██║   ██╔██╗ ██║█████╗   ╚███╔╝ ██║   ██║███████╗
   ██╔══██║██║   ██║██╔══╝  ██║╚██╗██║   ██║   ██║╚██╗██║██╔══╝   ██╔██╗ ██║   ██║╚════██║
   ██║  ██║╚██████╔╝███████╗██║ ╚████║   ██║   ██║ ╚████║███████╗██╔╝ ██╗╚██████╔╝███████║
   ╚═╝  ╚═╝ ╚═════╝ ╚══════╝╚═╝  ╚═══╝   ╚═╝   ╚═╝  ╚═══╝╚══════╝╚═╝  ╚═╝ ╚═════╝ ╚══════╝
                                         
                                          █████╗  ████╗  ███████╗
                                         ██╔══██╗ █  █  ╗██╔════╝
                                         ███████║  ██ ║  ███████╗
                                         ██╔══██║ █  █╔ ╚════██║
                                         ██║  ██║ ████  ╗███████║
                                         ╚═╝  ╚═╝ ╚══════╝╚══════╝`

	fmt.Print(teal)
	fmt.Println(banner)
	fmt.Printf("%sLet your agents Connect%s\n", boldTeal, reset)
	fmt.Printf("%sTalk to your agents, the suitable agent will take the mission%s\n\n", boldTeal, reset)
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

	tasks := []*types.Task{createInitialTask()}
	if err := app.orchestrator.ExecuteTasks(ctx, tasks); err != nil {
		return fmt.Errorf("error executing tasks: %w", err)
	}

	return nil
}
