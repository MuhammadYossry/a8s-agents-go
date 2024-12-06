# Python Code Agent

A powerful PydanticAI agent for automating Python code tasks including generation, improvement, testing, and deployment. Built with FastAPI and Pydantic for robust API interfaces and data validation.

## Features

- 🚀 **Code Generation**: Generate Python code from requirements
- 🔧 **Code Improvement**: Automatic code formatting and linting
- 🧪 **Testing**: Generate and run tests with coverage analysis
- 📦 **Deployment**: Preview deployment with environment configuration
- 📝 **Documentation**: Multiple documentation level support
- ✨ **Quality Assurance**: Built-in PEP8 compliance and code metrics

## Prerequisites

- Python 3.13+
- Poetry for dependency management
- Git (optional, for version control)

## Installation

1. Clone the repository:
```bash
git clone https://github.com/yourusername/python-code-agent.git
cd python-code-agent
```

2. Install poetry if you haven't already:
```bash
curl -sSL https://install.python-poetry.org | python3 -
```

3. Install dependencies:
```bash
poetry install
```

4. Activate the virtual environment:
```bash
poetry shell
```

## Running the Server

Start the FastAPI server using uvicorn:

```bash
poetry run uvicorn main:app --reload --port 8000
```

The API will be available at `http://localhost:8000`

## API Endpoints

### Generate Code

Generate Python code based on requirements.

```bash
curl -X POST http://localhost:8000/code_agent/python/generate_code \
  -H "Content-Type: application/json" \
  -d '{
    "code_requirements": {
      "description": "Create a REST API endpoint",
      "required_functions": ["get_user", "create_user"],
      "dependencies": ["fastapi", "sqlalchemy"],
      "python_version": "3.9",
      "testing_requirements": ["pytest"]
    },
    "style_guide": "PEP8",
    "include_tests": true,
    "documentation_level": "detailed"
  }'
```
response:
```json
{"generated_code": "\n            from fastapi import FastAPI\n            from pydantic import BaseModel\n\n            app = FastAPI()\n\n            \n                @app.get(\"/user/{user_id}\")\n                async def get_user(user_id: int):\n                    return {\"user_id\": user_id, \"message\": \"User retrieved\"}\n                \n\n                class UserCreate(BaseModel):\n                    username: str\n                    email: str\n\n                @app.post(\"/user/\")\n                async def create_user(user: UserCreate):\n                    return {\"username\": user.username, \"message\": \"User created\"}\n                \n            ",
  "description": "Generated code following PEP8 style guide",
  "test_cases": [
    "def test_get_user():\n    response = client.get('/user/1')\n    assert response.status_code == 200",
    "def test_create_user():\n    response = client.post('/user/', json={'username': 'test', 'email': 'test@example.com'})\n    assert response.status_code == 200"
  ],
  "documentation": {
    "overview": "Generated FastAPI endpoints for user management",
    "usage": "Run the server and access the endpoints via HTTP requests",
    "endpoints": "GET /user/{user_id}, POST /user/"
  }
}
```

### Improve Code

Improve existing code with formatting and linting.

```bash
curl -X POST http://localhost:8000/code_agent/python/improve_code \
  -H "Content-Type: application/json" \
  -d '{
    "changes_list": [{
      "file_path": "app/main.py",
      "original_code": "def hello_world():\n    print(\"Hello World\")\n",
      "proposed_changes": "def hello_world() -> str:\n    return \"Hello World\"\n",
      "change_type": "improvement",
      "priority": "medium"
    }],
    "apply_black_formatting": true,
    "run_linter": true
  }'
```

### Test Code

Generate and run tests for Python code.

```bash
curl -X POST http://localhost:8000/code_agent/python/test_code \
  -H "Content-Type: application/json" \
  -d '{
    "test_type": "unit",
    "require_passing": true,
    "test_instructions": "Test the hello_world function",
    "code_to_test": "def hello_world():\n    return \"Hello World\"\n",
    "minimum_coverage": 80.0
  }'
```

### Deploy Preview

Create a preview deployment of your code.

```bash
curl -X POST http://localhost:8000/deploy_agent/python/preview \
  -H "Content-Type: application/json" \
  -d '{
    "branch_id": "feature-branch-123",
    "is_private": true,
    "environment_vars": {
      "DEBUG": "true",
      "API_KEY": "test-key"
    }
  }'
```

## Configuration

The agent can be configured through environment variables:

```bash
export PYTHON_AGENT_HOST=0.0.0.0
export PYTHON_AGENT_PORT=8000
export PYTHON_AGENT_LOG_LEVEL=info
export PYTHON_AGENT_WORKERS=4
```

## Development

1. Create a new feature branch:
```bash
git checkout -b feature/your-feature-name
```

2. Run tests:
```bash
poetry run pytest
```

3. Format code:
```bash
poetry run black .
```

4. Run linter:
```bash
poetry run pylint python_code_agent
```

## API Documentation

Once the server is running, you can access:
- Interactive API documentation (Swagger UI): `http://localhost:8000/docs`
- Alternative API documentation (ReDoc): `http://localhost:8000/redoc`
- OpenAPI specification: `http://localhost:8000/openapi.json`

## Error Handling

The API uses standard HTTP status codes:
- `200`: Success
- `400`: Bad Request (invalid input)
- `404`: Not Found
- `500`: Internal Server Error

Detailed error messages are included in the response body:

```json
{
  "detail": "Error message describing what went wrong"
}
```

