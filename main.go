package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Relax-N-Tax/AgentNexus/types"
)

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

	tasks := []*types.Task{createInitialTask()}
	if err := app.orchestrator.ExecuteTasks(ctx, tasks); err != nil {
		log.Printf("error executing tasks: %v", err)
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
