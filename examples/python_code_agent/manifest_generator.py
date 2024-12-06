#    manifest_generator.py
from functools import wraps
from typing import List, Dict, Any, Optional, Type, Callable
from pydantic import BaseModel, Field
from fastapi import FastAPI
import inspect

class AgentMetadata(BaseModel):
    """Base model for agent endpoint metadata."""
    task_type: str
    skill_name: str
    description: str

class AgentEndpointRegistry:
    """Registry to store and manage agent endpoint metadata."""
    def __init__(self):
        self._endpoints: Dict[str, Dict[str, Any]] = {}
        self._base_url: str = "http://localhost:9200/v1"
        self._agent_id: str = "python-code-agent"
        self._description: str = "Advanced Python code generation, testing, and deployment agent"
        
        # Predefined skills and task types
        self._skills_by_type = {
            "pythonCodeTask": [
                "generateCode",
                "improveCode",
                "reviewCode",
                "formatCode"
            ],
            "pythonTestingTask": [
                "generateTests",
                "runTests",
                "analyzeCoverage"
            ],
            "pythonDeploymentTask": [
                "deployPreview",
                "configureEnvironment",
                "manageSecrets"
            ]
        }

    def register_endpoint(
        self,
        path: str,
        metadata: AgentMetadata,
        input_model: Type[BaseModel],
        output_model: Type[BaseModel]
    ):
        """Register an endpoint with its metadata and schema information."""
        self._endpoints[path] = {
            "metadata": metadata,
            "input_model": input_model,
            "output_model": output_model
        }

    def _extract_schema(self, model: Type[BaseModel], schema_type: str = "input") -> Dict[str, Any]:
        """Extract schema information from a Pydantic model."""
        schema = model.model_json_schema()
        if schema_type == "input":
            required = schema.get("required", [])
            properties = schema.get("properties", {})
            optional = [field for field in properties.keys() if field not in required]
            return {
                "type": "json",
                "required": required,
                "optional": optional
            }
        else:
            return {
                "type": "json",
                "fields": list(schema.get("properties", {}).keys())
            }

    def generate_config(self) -> Dict[str, List[Dict[str, Any]]]:
        """Generate the complete agent configuration."""
        actions = []
        for path, info in self._endpoints.items():
            metadata = info["metadata"]
            
            # Format the path to match the desired structure
            formatted_path = path.replace("/v1", "")
            
            actions.append({
                "name": metadata.skill_name,
                "path": formatted_path,
                "method": "POST",
                "inputSchema": self._extract_schema(info["input_model"], "input"),
                "outputSchema": self._extract_schema(info["output_model"], "output")
            })

        return {
            "agents": [{
                "id": self._agent_id,
                "type": "external",
                "description": self._description,
                "baseURL": self._base_url,
                "taskTypes": list(self._skills_by_type.keys()),
                "skillsByType": self._skills_by_type,
                "actions": actions,
                "features": {
                    "asyncProcessing": True,
                    "backgroundTasks": True,
                    "errorHandling": {
                        "retries": 3,
                        "fallbackBehavior": "graceful-degradation"
                    },
                    "security": {
                        "authentication": "required",
                        "authorization": "role-based"
                    }
                }
            }]
        }

# Global registry instance
registry = AgentEndpointRegistry()

def agent_endpoint(
    task_type: str,
    skill_name: str,
    description: str
) -> Callable:
    """Decorator to register an endpoint as an agent action."""
    def decorator(func: Callable) -> Callable:
        # Extract input and output models from function signature
        sig = inspect.signature(func)
        
        # Find the first Pydantic model parameter
        input_model = None
        for param in sig.parameters.values():
            if hasattr(param.annotation, 'model_json_schema'):  # Check for Pydantic model
                input_model = param.annotation
                break
        
        # Get the output model from the return annotation
        output_model = None
        if hasattr(func, '__annotations__') and 'return' in func.__annotations__:
            return_type = func.__annotations__['return']
            # Handle async return types
            if hasattr(return_type, '__origin__') and return_type.__origin__ is None:
                return_type = return_type.__args__[0]
            output_model = return_type

        if not input_model or not output_model:
            raise ValueError(f"Both input and output models must be Pydantic models for {func.__name__}")
            
        # Get default path from function name if closure is not available
        path = f"/v1/code_agent/python/{func.__name__}"

        # Register the endpoint
        metadata = AgentMetadata(
            task_type=task_type,
            skill_name=skill_name,
            description=description
        )
        
        registry.register_endpoint(path, metadata, input_model, output_model)

        @wraps(func)
        async def wrapper(*args, **kwargs):
            return await func(*args, **kwargs)

        return wrapper
    return decorator


def setup_agent_routes(app: FastAPI):
    """Set up the agents.json endpoint."""
    @app.get("/agents.json")
    async def get_agent_config():
        return registry.generate_config()