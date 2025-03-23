# AI Task Routing System
# AgentNexus (a8s)

> ⚠️ **Experimental Project**: This is an experimental learning project exploring LLM agents and task routing. Not recommended for production use.

## Overview

AgentNexus (a8s) is an experimental Go-based project for routing and orchestrating AI tasks across multiple specialized AI and LLM agents hosted remotely. It demonstrates concepts around AI task distribution, agent collaboration, and dynamic capability matching.
![alt text](https://pbs.twimg.com/media/GZhrK4IWcAIb949?format=jpg&name=medium)
![image](https://github.com/user-attachments/assets/55b62a8d-4984-4336-9ff7-f99789a80b51)


## Key Features

- **Dynamic Task Routing**: Routes tasks to the most capable AI agent based on skill requirements
- **Agent Hub System**: Central registry for managing and deploying AI agents with initial versioning support
- **Capability-Based Matching**: Matches tasks to agents using a basic capability scoring system
- **SQLite-Based Registry**: Persistent storage for agent definitions with version control
- **Python Agent Examples**: Includes example agents for code generation, RAG (Retrieval-Augmented Generation), and more

## Project Status

This project is currently in an **experimental phase** and serves several purposes:
- Research into LLM-based task routing and agent collaboration
- Learning and implementing Go practices
- Exploring agent-based architectures for AI systems

### Current Limitations

- Not production-ready
- Limited error handling in some areas
- Experimental implementation of agent workflows
- Some features are partially implemented
- Documentation needs improvement

## Interesting Technical Aspects

1. **Capability Matching System**
   - Sophisticated scoring mechanism for matching tasks to agents
   - Hierarchical skill path evaluation
   - Dynamic capability registration

2. **Agent Hub Architecture**
   - Version-controlled agent registry
   - RESTful API for agent management
   - Support for different agent types (internal/external)

3. **Task Extraction and Routing**
   - LLM-based task analysis
   - Dynamic routing based on agent capabilities
   - Extensible agent interface system

## Getting Started

### Prerequisites
- Go 1.21+
- Python 3.9+ (for example agents)
- SQLite3

### Basic Setup
```bash
# Clone the repository
git clone https://github.com/yourusername/AgentNexus

# Install Go dependencies
go mod tidy

# Configure LLM settings
cp a8s_llm.conf.example a8s_llm.conf
# Edit a8s_llm.conf with your LLM settings
# Add agents to load from the registry
cp a8s_agents.conf.example a8s.conf.example
# Start the system
go run .
```

## Project Structure Highlights

```
AgentNexus/
├── hub/           # Agent registry and management
├── orchestrator/  # Task orchestration and routing
├── capability/    # Capability matching system
├── examples/      # Example agents and implementations
└── internal/      # Core agent implementations
```

## Learning Value

This project is particularly useful for:
- Learning about AI distributed agents and how we can build task routing systems
- Exploring LLM-based agent systems and capability-based service matching

## Contributing

While this is primarily a research project, contributions and discussions are welcome! Feel free to:
- Submit PRs for improvements
- Share ideas about agent architectures
- Report bugs or suggest features

## Disclaimer

This project is meant for learning and experimenting purposes. It demonstrates concepts around AI task routing and agent systems but should not be used in production environments as it is still in early development

## License

This project is licensed under the MIT License
