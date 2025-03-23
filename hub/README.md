# AgentsHub - Agent Registry and Distribution System

AgentsHub is a lightweight registry system for managing and distributing AI agent definition(Agentfile), similar to how container registries works for container images(think of the popular Dockerfile). It provides a simple way to store, version, and retrieve agent definition through both an HTTP API and CLI tool.

## Features

- Push and pull agent configuration files
- Version control for agents definitions
- Simple HTTP API for integration
- Command-line interface (CLI) for easy management
- In-memory storage (with extensible storage backend)


## Usage

### Starting the Server

The AgentsHub server is integrated into the main application:

```bash
go run main.go
```

This will start both the orchestrator and the AgentsHub server on port 8082.

### Using the CLI

The `a8shub` CLI tool provides commands for interacting with the registry.

#### Basic Commands

```bash
# Push an agent definition
a8shub push       # Example: a8shub push myagent:1.0 agent.json
# Output: Successfully pushed myagent:1.0

# Pull an agent definition
a8shub pull             # Example: a8shub pull myagent:1.0
# Output: Successfully pulled myagent:1.0 to myagent-1.0.json

# Show agent documentation
a8shub show           # Example: a8shub show myagent:1.0
                                      # or: a8shub show myagent (shows latest version)
```

#### Version Format
The system uses semantic versioning (SemVer):
- Accepts versions like: "1.0", "v1.0", "1.0.0", "v1.0.0"
- When version is omitted in 'show' command, automatically uses latest version
- Latest version is determined by highest numerical value

#### CLI Options

```bash
# Use a different server with different port
a8shub --server http://other-server:8090 push myagent:1.0 config.md

# Get help
a8shub --help
a8shub push --help
```

## API Endpoints

### Push Agent Definition

```http
POST /v1/push
Content-Type: multipart/form-data

Form fields:
- agentfile: The agent definition file
- name: Agent name
- version: Agent version
```

### Pull Agent Definition

```http
GET /v1/pull?name=<name>&version=<version>
```


### Show Agent Documentation

```http
GET /v1/show?name=&version=
```
Response: Markdown formatted documentation of the agent definition

## Development

The system is designed to be extensible:

1. The `Registry` interface can be implemented for different storage backends
2. New API endpoints can be added to the server
3. CLI commands can be extended using Cobra

## Configuration

Default configuration:

- Server port: 8082
- Server URL: http://localhost:8082
- Max file size: 10MB

## Best Practices

1. Use semantic versioning for agent configurations
2. Include proper documentation in your agent files
3. Use meaningful names for your agents
4. Test configurations before pushing to production registries

## Security

Current limitations:
- No authentication/authorization
- In-memory storage only
- No SSL/TLS support

For production use, consider implementing these security features.