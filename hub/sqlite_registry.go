package hub

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/MuhammadYossry/a8s-agents-go/types"
	_ "github.com/mattn/go-sqlite3"
)

type SQLiteRegistry struct {
	db     *sql.DB
	mutex  sync.RWMutex
	logger *log.Logger
}

var (
	registryInstance *SQLiteRegistry
	registryOnce     sync.Once
	registryErr      error
)

func GetSQLiteRegistry() (*SQLiteRegistry, error) {
	registryOnce.Do(func() {
		registryInstance, registryErr = initSQLiteRegistry()
	})
	return registryInstance, registryErr
}

func initSQLiteRegistry() (*SQLiteRegistry, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory: %w", err)
	}

	dbDir := filepath.Join(cwd, ".a8s")
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	dbPath := filepath.Join(dbDir, "a8s-hub.db")
	logger := log.New(log.Writer(), "[AgentsHub] ", log.LstdFlags|log.Lshortfile)
	logger.Printf("Using SQLite database at: %s", dbPath)

	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	schema := `
    CREATE TABLE IF NOT EXISTS agents (
        name TEXT NOT NULL,
        version TEXT NOT NULL,
        data JSON NOT NULL,
		updated_at INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
        created_at INTEGER NOT NULL,
        PRIMARY KEY (name, version)
    );`

	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create schema: %w", err)
	}

	reg := &SQLiteRegistry{db: db, logger: logger}
	if err := reg.ListAgents(); err != nil {
		logger.Printf("Warning: Failed to list agents: %v", err)
	}

	return reg, nil
}

func (r *SQLiteRegistry) Store(name, version string, agent *AgentFile) error {
	v, err := ParseVersion(version)
	if err != nil {
		return fmt.Errorf("invalid version format: %v", err)
	}

	data, err := json.Marshal(agent)
	if err != nil {
		return fmt.Errorf("failed to marshal agent data: %w", err)
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	query := `
    INSERT OR REPLACE INTO agents (name, version, data, created_at, updated_at)
    VALUES (?, ?, ?, ?, strftime('%s', 'now'))
    `

	_, err = r.db.Exec(query, name, v.String(), string(data), agent.CreateTime)
	if err != nil {
		return fmt.Errorf("failed to store agent: %w", err)
	}

	return nil
}

func (r *SQLiteRegistry) Get(name, version string) (*AgentFile, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// First check if the agent exists at all
	var count int
	err := r.db.QueryRow("SELECT COUNT(*) FROM agents WHERE name = ?", name).Scan(&count)
	if err != nil {
		return nil, fmt.Errorf("failed to check agent existence: %w", err)
	}
	if count == 0 {
		return nil, fmt.Errorf("agent %s not found", name)
	}

	// If version is empty or "latest", find the highest version
	if version == "" {
		version = "latest"
	}
	if version == "latest" {
		return r.getLatestVersion(name)
	}

	// Normalize version format
	normalizedVersion := version
	if !strings.HasPrefix(normalizedVersion, "v") {
		normalizedVersion = "v" + normalizedVersion
	}

	// Add .0 suffix if needed to match semantic versioning
	if strings.Count(normalizedVersion, ".") == 0 {
		normalizedVersion += ".0.0"
	} else if strings.Count(normalizedVersion, ".") == 1 {
		normalizedVersion += ".0"
	}

	var data string
	query := `
    SELECT data FROM agents
    WHERE name = ?
    AND version IN (?, ?, ?, ?)
    `

	// Try all possible version formats
	shortVersion := strings.TrimSuffix(normalizedVersion, ".0")
	shortVersion = strings.TrimSuffix(shortVersion, ".0")
	veryShortVersion := strings.TrimPrefix(shortVersion, "v")

	err = r.db.QueryRow(
		query,
		name,
		normalizedVersion,
		shortVersion,
		veryShortVersion,
		strings.TrimPrefix(normalizedVersion, "v"),
	).Scan(&data)

	if err == sql.ErrNoRows {
		// Remove debug logging here, just collect versions
		rows, _ := r.db.Query("SELECT version FROM agents WHERE name = ?", name)
		var versions []string
		for rows.Next() {
			var v string
			rows.Scan(&v)
			versions = append(versions, v)
		}
		rows.Close()

		return nil, fmt.Errorf("agent %s:%s not found (looked for: %s, %s, %s, %s). Available: %v",
			name, version, normalizedVersion, shortVersion, veryShortVersion,
			strings.TrimPrefix(normalizedVersion, "v"), versions)
	}
	if err != nil {
		return nil, nil
	}

	var agent AgentFile
	if err := json.Unmarshal([]byte(data), &agent); err != nil {
		return nil, fmt.Errorf("failed to parse agent data: %w", err)
	}

	r.logger.Printf("Successfully retrieved agent %s:%s", name, version)
	return &agent, nil
}

func (r *SQLiteRegistry) GetJSON(name, version string) ([]byte, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	if version == "" || version == "latest" {
		agent, err := r.getLatestVersion(name)
		if err != nil {
			return nil, err
		}
		return json.Marshal(agent)
	}

	v, err := ParseVersion(version)
	if err != nil {
		return nil, fmt.Errorf("invalid version format: %v", err)
	}

	var data string
	query := `SELECT data FROM agents WHERE name = ? AND version = ?`
	err = r.db.QueryRow(query, name, v.String()).Scan(&data)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("version %s not found for agent %s", version, name)
	}
	if err != nil {
		return nil, nil
	}

	return []byte(data), nil
}

func (r *SQLiteRegistry) getLatestVersion(name string) (*AgentFile, error) {
	// First check if the agent exists
	var count int
	err := r.db.QueryRow("SELECT COUNT(*) FROM agents WHERE name = ?", name).Scan(&count)
	if err != nil {
		return nil, fmt.Errorf("failed to check agent existence: %w", err)
	}
	if count == 0 {
		return nil, fmt.Errorf("agent %s not found", name)
	}

	query := `
    SELECT data FROM agents
    WHERE name = ?
    ORDER BY
        CAST(REPLACE(REPLACE(version, 'v', ''), '.', '') AS INTEGER) DESC
    LIMIT 1
    `

	var data string
	err = r.db.QueryRow(query, name).Scan(&data)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("no versions found for agent %s", name)
	}
	if err != nil {
		return nil, nil
	}

	var agent AgentFile
	if err := json.Unmarshal([]byte(data), &agent); err != nil {
		return nil, fmt.Errorf("failed to parse agent data: %w", err)
	}

	return &agent, nil
}

func (r *SQLiteRegistry) ListAgents() error {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	query := `SELECT name, version, created_at, updated_at FROM agents`
	rows, err := r.db.Query(query)
	if err != nil {
		return fmt.Errorf("failed to query agents: %w", err)
	}
	defer rows.Close()

	r.logger.Println("Available agents in registry:")
	for rows.Next() {
		var name, version string
		var createdAt, updatedAt int64
		if err := rows.Scan(&name, &version, &createdAt, &updatedAt); err != nil {
			return fmt.Errorf("failed to scan row: %w", err)
		}
		r.logger.Printf("- %s:%s (created: %s, updated: %s)",
			name, version,
			time.Unix(createdAt, 0).Format(time.RFC3339),
			time.Unix(updatedAt, 0).Format(time.RFC3339))
	}

	return nil
}

func (r *SQLiteRegistry) GetAgentDef(name, version string) (*types.AgentDefinition, error) {
	agentFile, err := r.Get(name, version)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent file: %w", err)
	}

	var agentDef types.AgentDefinition
	if err := json.Unmarshal([]byte(agentFile.Content), &agentDef); err != nil {
		return nil, fmt.Errorf("failed to parse agent definition: %w", err)
	}

	// Validate required fields
	if agentDef.ID == "" || agentDef.BaseURL == "" {
		return nil, fmt.Errorf("invalid agent definition: missing required fields")
	}

	return &agentDef, nil
}

func (r *SQLiteRegistry) Close() error {
	return r.db.Close()
}
