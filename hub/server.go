// File: hub/server.go
package hub

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// Server handles HTTP requests for the AgentsHub
type Server struct {
	registry Registry
	server   *http.Server
	config   Config
	logger   *log.Logger
}

// NewServer creates a new AgentsHub server
func NewServer(config Config, registry Registry) *Server {
	if registry == nil {
		registry = NewInMemoryRegistry()
	}

	return &Server{
		registry: registry,
		config:   config,
		logger:   log.New(log.Writer(), "[AgentsHub] ", log.LstdFlags|log.Lshortfile),
	}
}

func (s *Server) Start(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/push", s.handlePush)
	mux.HandleFunc("/v1/pull", s.handlePull)

	s.server = &http.Server{
		Addr:    s.config.Address,
		Handler: mux,
	}

	s.logger.Printf("Starting AgentsHub server on %s", s.config.Address)

	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Printf("Server error: %v", err)
		}
	}()

	// Wait a moment to ensure server is up
	time.Sleep(100 * time.Millisecond)
	s.logger.Printf("AgentsHub server is ready")

	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Printf("Shutting down AgentsHub server...")

	if s.server != nil {
		if err := s.server.Shutdown(ctx); err != nil {
			return fmt.Errorf("error shutting down server: %v", err)
		}
	}

	if err := s.registry.Close(); err != nil {
		return fmt.Errorf("error closing registry: %v", err)
	}

	s.logger.Printf("AgentsHub server shutdown complete")
	return nil
}

func (s *Server) handlePush(w http.ResponseWriter, r *http.Request) {
	s.logger.Printf("Received push request from %s", r.RemoteAddr)

	if r.Method != http.MethodPost {
		s.logger.Printf("Method not allowed: %s", r.Method)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseMultipartForm(10 << 20) // 10 MB max
	if err != nil {
		s.logger.Printf("Failed to parse form: %v", err)
		http.Error(w, "failed to parse form", http.StatusBadRequest)
		return
	}

	file, handler, err := r.FormFile("agentfile")
	if err != nil {
		s.logger.Printf("Failed to get file: %v", err)
		http.Error(w, "failed to get file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		s.logger.Printf("Failed to read file: %v", err)
		http.Error(w, "failed to read file", http.StatusInternalServerError)
		return
	}

	name := r.FormValue("name")
	version := r.FormValue("version")
	s.logger.Printf("Processing agent %s:%s", name, version)

	if name == "" || version == "" {
		s.logger.Printf("Missing name or version")
		http.Error(w, "name and version required", http.StatusBadRequest)
		return
	}

	agent := &AgentFile{
		Name:       name,
		Version:    version,
		Content:    string(content),
		Metadata:   map[string]string{"filename": handler.Filename},
		CreateTime: time.Now().Unix(),
	}

	err = s.registry.Store(name, version, agent)
	if err != nil {
		s.logger.Printf("Failed to store agent: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.logger.Printf("Successfully stored agent %s:%s", name, version)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": fmt.Sprintf("Agent %s:%s pushed successfully", name, version),
	})
}

func (s *Server) handlePull(w http.ResponseWriter, r *http.Request) {
	s.logger.Printf("Received pull request from %s", r.RemoteAddr)

	if r.Method != http.MethodGet {
		s.logger.Printf("Method not allowed: %s", r.Method)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	name := r.URL.Query().Get("name")
	version := r.URL.Query().Get("version")
	s.logger.Printf("Processing pull request for %s:%s", name, version)

	if name == "" || version == "" {
		s.logger.Printf("Missing name or version")
		http.Error(w, "name and version required", http.StatusBadRequest)
		return
	}

	agent, err := s.registry.Get(name, version)
	if err != nil {
		s.logger.Printf("Failed to get agent: %v", err)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	s.logger.Printf("Successfully retrieved agent %s:%s", name, version)

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s-%s.md", name, version))
	w.Header().Set("Content-Type", "text/markdown")

	fmt.Fprint(w, agent.Content)
}
