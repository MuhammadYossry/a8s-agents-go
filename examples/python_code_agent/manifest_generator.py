from functools import wraps
from typing import List, Dict, Any, Optional, Type, Callable, Literal
from pydantic import BaseModel, Field
from fastapi import FastAPI
import inspect
from urllib.parse import urljoin

class CapabilityMetadata(BaseModel):
    expertise: Optional[str] = None
    versions: Optional[List[str]] = None
    frameworks: Optional[List[str]] = None
    tools: Optional[List[str]] = None
    platforms: Optional[List[str]] = None

class Capability(BaseModel):
    skillPath: List[str]
    level: Literal["domain", "specialty", "skill"]
    metadata: CapabilityMetadata

class AgentMetadata(BaseModel):
    task_type: str
    skill_name: str
    description: str

class AgentEndpointRegistry:
    def __init__(self):
        self._endpoints: Dict[str, Dict[str, Any]] = {}
        self._base_url: Optional[str] = None
        self._agent_id: Optional[str] = None
        self._description: Optional[str] = None
        self._capabilities: List[Capability] = []
        self._mount_path: Optional[str] = None

    def configure(
        self,
        base_url: str,
        agent_id: str,
        description: str,
        capabilities: List[Capability],
        mount_path: Optional[str] = None
    ) -> None:
        """Configure the registry with agent-level information."""
        self._base_url = base_url.rstrip('/')
        self._agent_id = agent_id
        self._description = description
        self._capabilities = capabilities
        self._mount_path = mount_path.strip('/') if mount_path else None

    def _normalize_path(self, path: str) -> str:
        """Normalize API path to ensure consistent formatting."""
        # Remove leading/trailing whitespace and ensure single leading slash
        path = '/' + path.strip('/').strip()
        
        # Handle mount path if present
        if self._mount_path:
            path = f"/{self._mount_path}{path}"
        
        # Replace multiple consecutive slashes with single slash
        while '//' in path:
            path = path.replace('//', '/')
            
        return path

    def register_endpoint(
        self,
        path: str,
        metadata: AgentMetadata,
        input_model: Type[BaseModel],
        output_model: Type[BaseModel],
        examples: Optional[Dict[str, List[Dict[str, Any]]]] = None
    ) -> None:
        """Register an endpoint with normalized path."""
        normalized_path = self._normalize_path(path)
        self._endpoints[normalized_path] = {
            "metadata": metadata,
            "input_model": input_model,
            "output_model": output_model,
            "examples": examples or {"validRequests": []}
        }

    def _extract_schema(self, model: Type[BaseModel]) -> Dict[str, Any]:
        """Extract and format schema information."""
        schema = model.model_json_schema()
        required = schema.get("required", [])
        # Ensure required field is always a list
        if required is None:
            required = []
        return {
            "type": "object",
            "required": required,
            "properties": schema.get("properties", {})
        }

    def generate_config(self) -> Dict[str, List[Dict[str, Any]]]:
        """Generate the complete agent configuration."""
        if not all([self._base_url, self._agent_id, self._description]):
            raise ValueError("Registry not properly configured. Call configure() first.")

        actions = []
        for path, info in self._endpoints.items():
            metadata = info["metadata"]
            actions.append({
                "name": metadata.skill_name,
                "path": path,
                "method": "POST",
                "inputSchema": self._extract_schema(info["input_model"]),
                "outputSchema": self._extract_schema(info["output_model"]),
                "examples": info["examples"]
            })

        return {
            "agents": [{
                "id": self._agent_id,
                "type": "external",
                "description": self._description,
                "baseURL": self._base_url,
                "capabilities": [cap.dict(exclude_none=True) for cap in self._capabilities],
                "actions": sorted(actions, key=lambda x: x["path"])  # Sort for consistency
            }]
        }

# Global registry instance
registry = AgentEndpointRegistry()

def configure_agent(
    base_url: str,
    agent_id: str,
    description: str,
    capabilities: List[Capability],
    mount_path: Optional[str] = None
):
    """Configure agent-level attributes."""
    def decorator(app: FastAPI):
        registry.configure(
            base_url=base_url,
            agent_id=agent_id,
            description=description,
            capabilities=capabilities,
            mount_path=mount_path
        )
        return app
    return decorator

def agent_endpoint(
    task_type: str,
    skill_name: str,
    description: str,
    examples: Optional[Dict[str, List[Dict[str, Any]]]] = None
) -> Callable:
    """Register an endpoint as an agent action."""
    def decorator(func: Callable) -> Callable:
        sig = inspect.signature(func)
        
        # Find input/output models
        input_model = next(
            (param.annotation for param in sig.parameters.values() 
             if hasattr(param.annotation, 'model_json_schema')),
            None
        )
        
        return_annotation = func.__annotations__.get('return')
        output_model = None
        if hasattr(return_annotation, '__origin__'):
            output_model = return_annotation.__args__[0]
        else:
            output_model = return_annotation

        if not input_model or not output_model:
            raise ValueError(
                f"Both input and output models must be Pydantic models for {func.__name__}"
            )

        metadata = AgentMetadata(
            task_type=task_type,
            skill_name=skill_name,
            description=description
        )

        # Store metadata for registration
        func._agent_metadata = metadata
        func._agent_input_model = input_model
        func._agent_output_model = output_model
        func._agent_examples = examples

        @wraps(func)
        async def wrapper(*args, **kwargs):
            return await func(*args, **kwargs)

        return wrapper
    return decorator

def setup_agent_routes(app: FastAPI) -> None:
    """Set up the agents.json endpoint and register all routes."""
    def register_routes(routes, prefix=""):
        for route in routes:
            # Check if this is a mounted application
            if isinstance(getattr(route, "app", None), FastAPI):
                # Handle mounted apps recursively
                mounted_prefix = prefix + str(route.path).rstrip("/")
                register_routes(route.app.routes, mounted_prefix)
            # Check for regular endpoints with agent metadata
            elif hasattr(route, "endpoint") and hasattr(route.endpoint, "_agent_metadata"):
                func = route.endpoint
                route_path = prefix + str(route.path)
                registry.register_endpoint(
                    path=route_path,
                    metadata=func._agent_metadata,
                    input_model=func._agent_input_model,
                    output_model=func._agent_output_model,
                    examples=func._agent_examples
                )

    register_routes(app.routes)

    @app.get("/agents.json")
    async def get_agent_config():
        return registry.generate_config()