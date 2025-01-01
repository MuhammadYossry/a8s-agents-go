from functools import wraps
from typing import List, Dict, Any, Optional, Type, Callable, Literal, Set, Union
from pydantic import BaseModel, Field
from fastapi import FastAPI
import inspect
from enum import Enum
from datetime import datetime
import warnings

class ActionType(str, Enum):
    """Types of actions an agent can perform."""
    TALK = "talk"  # For conversation/dialogue actions
    GENERATE = "generate"  # For content generation actions

class BaseSchemaModel(BaseModel):
    """Base model for all schema definitions with common configuration."""
    class Config:
        json_schema_extra = {"examples": []}

class CapabilityMetadata(BaseModel):
    """Capability metadata definition."""
    expertise: Optional[str] = None
    versions: Optional[List[str]] = None
    frameworks: Optional[List[str]] = None
    tools: Optional[List[str]] = None
    platforms: Optional[List[str]] = None

class Capability(BaseModel):
    """Capability definition."""
    skillPath: List[str]
    level: Literal["domain", "specialty", "skill"]
    metadata: CapabilityMetadata

class ActionMetadata(BaseModel):
    """Metadata for agent actions."""
    action_type: ActionType
    name: str
    description: str
    task_type: Optional[str] = None  # For backward compatibility

class ActionEndpointInfo(BaseModel):
    """Information about an action endpoint including its schema definitions."""
    metadata: ActionMetadata
    input_model: Type[BaseModel]
    output_model: Type[BaseModel]
    schema_definitions: Optional[Dict[str, Type[BaseModel]]] = None
    examples: Optional[Dict[str, List[Dict[str, Any]]]] = None
    route_path: str

class AgentRegistry:
    """Registry for agent actions and their endpoints."""
    def __init__(self):
        self._action_endpoints: Dict[str, ActionEndpointInfo] = {}
        self._base_url: Optional[str] = None
        self._agent_id: Optional[str] = None
        self._description: Optional[str] = None
        self._capabilities: List[Capability] = []
        self._mount_path: Optional[str] = None
        self._schema_definitions: Dict[str, Type[BaseModel]] = {}

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
        path = '/' + path.strip('/').strip()
        if self._mount_path:
            path = f"/{self._mount_path}{path}"
        while '//' in path:
            path = path.replace('//', '/')
        return path

    def register_action_endpoint(
        self,
        path: str,
        endpoint_info: ActionEndpointInfo,
    ) -> None:
        """Register an action endpoint with its schema definitions."""
        normalized_path = self._normalize_path(path)
        endpoint_info.route_path = normalized_path
        self._action_endpoints[normalized_path] = endpoint_info
        
        if endpoint_info.schema_definitions:
            self._schema_definitions.update(endpoint_info.schema_definitions)

    def _extract_schema(self, model: Type[BaseModel]) -> Dict[str, Any]:
        """Extract schema information with proper reference handling."""
        schema = model.model_json_schema()
        refs = self._find_schema_refs(schema)
        
        for ref_name in refs:
            if ref_name in self._schema_definitions:
                ref_model = self._schema_definitions[ref_name]
                if "$defs" not in schema:
                    schema["$defs"] = {}
                schema["$defs"][ref_name] = ref_model.model_json_schema()
        
        return schema

    def _find_schema_refs(self, schema: Dict[str, Any]) -> Set[str]:
        """Find all schema references in a given schema."""
        refs = set()
        
        def find_refs(obj: Any):
            if isinstance(obj, dict):
                if "$ref" in obj:
                    ref = obj["$ref"].split("/")[-1]
                    refs.add(ref)
                for value in obj.values():
                    find_refs(value)
            elif isinstance(obj, list):
                for item in obj:
                    find_refs(item)
                    
        find_refs(schema)
        return refs

    def _format_action_endpoint(self, info: ActionEndpointInfo) -> Dict[str, Any]:
        """Format action endpoint information for output."""
        return {
            "name": info.metadata.name,
            "actionType": info.metadata.action_type,
            "path": info.route_path,
            "method": "POST",
            "inputSchema": self._extract_schema(info.input_model),
            "outputSchema": self._extract_schema(info.output_model),
            "examples": info.examples or {"validRequests": []},
            "description": info.metadata.description
        }

    def generate_config(self) -> Dict[str, Any]:
        """Generate the complete agent configuration."""
        if not all([self._base_url, self._agent_id, self._description]):
            raise ValueError("Registry not properly configured. Call configure() first.")

        config = {
            "agents": [{
                "id": self._agent_id,
                "type": "external",
                "description": self._description,
                "baseURL": self._base_url,
                "capabilities": [cap.model_dump(exclude_none=True) for cap in self._capabilities],
                "actions": sorted(
                    [self._format_action_endpoint(info) 
                     for info in self._action_endpoints.values()],
                    key=lambda x: x["path"]
                )
            }]
        }

        if self._schema_definitions:
            config["$defs"] = {
                name: model.model_json_schema()
                for name, model in self._schema_definitions.items()
            }

        return config

# Global registry instance
registry = AgentRegistry()

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

def agent_action(
    action_type: ActionType,
    name: str,
    description: str,
    schema_definitions: Optional[Dict[str, Type[BaseModel]]] = None,
    examples: Optional[Dict[str, List[Dict[str, Any]]]] = None
) -> Callable:
    """Register an endpoint as an agent action."""
    def decorator(func: Callable) -> Callable:
        sig = inspect.signature(func)
        
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

        endpoint_info = ActionEndpointInfo(
            metadata=ActionMetadata(
                action_type=action_type,
                name=name,
                description=description
            ),
            input_model=input_model,
            output_model=output_model,
            schema_definitions=schema_definitions,
            examples=examples,
            route_path=""  # Will be set during registration
        )

        func._endpoint_info = endpoint_info

        @wraps(func)
        async def wrapper(*args, **kwargs):
            return await func(*args, **kwargs)

        return wrapper
    return decorator

# Backward compatibility layer
def agent_endpoint(
    task_type: str,
    skill_name: str,
    description: str,
    schema_definitions: Optional[Dict[str, Type[BaseModel]]] = None,
    examples: Optional[Dict[str, List[Dict[str, Any]]]] = None
) -> Callable:
    """
    Legacy decorator for backward compatibility.
    Warns about deprecation and maps to new agent_action decorator.
    """
    warnings.warn(
        "agent_endpoint is deprecated. Use agent_action instead.",
        DeprecationWarning,
        stacklevel=2
    )
    
    # Map task_type to ActionType (basic mapping, expand as needed)
    action_type_map = {
        "pythonCodeTask": ActionType.GENERATE,
        "pythonTestingTask": ActionType.GENERATE,
        "pythonDeploymentTask": ActionType.GENERATE,
        # Add more mappings as needed
    }
    
    action_type = action_type_map.get(task_type, ActionType.GENERATE)
    
    return agent_action(
        action_type=action_type,
        name=skill_name,
        description=description,
        schema_definitions=schema_definitions,
        examples=examples
    )

def setup_agent_routes(app: FastAPI) -> None:
    """Set up the agents.json endpoint and register all routes."""
    def register_routes(routes, prefix=""):
        for route in routes:
            if isinstance(getattr(route, "app", None), FastAPI):
                mounted_prefix = prefix + str(route.path).rstrip("/")
                register_routes(route.app.routes, mounted_prefix)
            elif hasattr(route, "endpoint") and hasattr(route.endpoint, "_endpoint_info"):
                func = route.endpoint
                route_path = prefix + str(route.path)
                registry.register_action_endpoint(
                    path=route_path,
                    endpoint_info=func._endpoint_info
                )
    
    register_routes(app.routes)

    @app.get("/agents.json")
    async def get_agent_config():
        return registry.generate_config()