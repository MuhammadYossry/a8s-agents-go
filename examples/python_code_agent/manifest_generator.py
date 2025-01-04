from functools import wraps
from typing import List, Dict, Any, Optional, Type, Callable, Literal, Set, Union
from pydantic import BaseModel, ConfigDict, Field
from fastapi import FastAPI, Response, HTTPException
from fastapi.responses import HTMLResponse
import inspect
from enum import Enum
from pathlib import Path
import re

def slugify(text: str) -> str:
    """Convert text to URL-safe slug."""
    text = text.lower()
    text = re.sub(r'[^\w\s-]', '', text)
    text = re.sub(r'[-\s]+', '-', text)
    return text.strip('-')

class ActionType(str, Enum):
    """Types of actions an agent can perform."""
    TALK = "talk"  # For conversation/dialogue actions
    GENERATE = "generate"  # For content generation actions

class CapabilityMetadata(BaseModel):
    """Metadata for agent capabilities."""
    expertise: Optional[str] = None
    versions: Optional[List[str]] = None
    frameworks: Optional[List[str]] = None
    tools: Optional[List[str]] = None
    platforms: Optional[List[str]] = None

class Capability(BaseModel):
    """Capability definition matching FastAPI model."""
    skill_path: List[str] = Field(..., alias="skillPath")
    level: Literal["domain", "specialty", "skill"]
    meta_info: CapabilityMetadata = Field(..., alias="metadata")
    
    model_config = ConfigDict(
        populate_by_name=True,
        json_schema_extra={
            "examples": [{
                "skillPath": ["Development"],
                "level": "domain",
                "metadata": {"expertise": "advanced"}
            }]
        }
    )

class ActionMetadata(BaseModel):
    """Metadata for agent actions."""
    action_type: ActionType
    name: str
    description: str

class ActionEndpointInfo(BaseModel):
    """Information about an action endpoint."""
    metadata: ActionMetadata
    input_model: Type[BaseModel]
    output_model: Type[BaseModel]
    schema_definitions: Optional[Dict[str, Type[BaseModel]]] = None
    examples: Optional[Dict[str, List[Dict[str, Any]]]] = None
    route_path: str

class AgentRegistry:
    """Registry for managing agent configuration and endpoints."""
    
    def __init__(self):
        self._action_endpoints: Dict[str, ActionEndpointInfo] = {}
        self._base_url: Optional[str] = None
        self._name: Optional[str] = None
        self._description: Optional[str] = None
        self._capabilities: List[Capability] = []
        self._schema_definitions: Dict[str, Type[BaseModel]] = {}
        self._version: Optional[str] = None
        self._mount_path: Optional[str] = None
        self._html_path: Optional[str] = None
        self._json_path: Optional[str] = None
        self._dashboard_template_path: Optional[Path] = None
        self._registered_agents: List[Dict[str, str]] = []

    def configure(
        self,
        base_url: str,
        name: str,
        version: str,
        description: str,
        capabilities: List[Capability],
        mount_path: Optional[str] = None,
        html_path: Optional[str] = None,
        json_path: Optional[str] = None,
        dashboard_template_path: Optional[Union[str, Path]] = None
    ) -> None:
        """Configure the agent registry with required attributes."""
        self._base_url = base_url.rstrip('/')
        self._name = name
        self._version = version
        self._description = description
        self._capabilities = capabilities
        self._mount_path = mount_path.strip('/') if mount_path else None
        self._html_path = html_path
        self._json_path = json_path
        
        # Register the agent for agents.json listing
        agent_slug = slugify(name)
        json_endpoint = json_path if json_path else f"/agents/{agent_slug}.json"
        html_endpoint = html_path if html_path else f"/agents/{agent_slug}"
        
        self._registered_agents.append({
            "name": name,
            "slug": agent_slug,
            "version": version,
            "manifestUrl": f"{base_url}{json_endpoint}",
            "dashboardUrl": f"{base_url}{html_endpoint}"
        })
        
        if dashboard_template_path:
            self._dashboard_template_path = Path(dashboard_template_path)
            if not self._dashboard_template_path.exists():
                raise FileNotFoundError(f"Agent dashboard template not found at {dashboard_template_path}")

    def _normalize_path(self, path: str) -> str:
        """Normalize API path with proper prefix."""
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
        """Register an action endpoint with its schema."""
        normalized_path = self._normalize_path(path)
        endpoint_info.route_path = normalized_path
        self._action_endpoints[normalized_path] = endpoint_info
        
        if endpoint_info.schema_definitions:
            self._schema_definitions.update(endpoint_info.schema_definitions)

    def get_agent_dashboard(self) -> Optional[str]:
        """Return the agent dashboard HTML if template exists."""
        if not self._dashboard_template_path:
            return None
            
        try:
            return self._dashboard_template_path.read_text(encoding='utf-8')
        except Exception as e:
            raise ValueError(f"Failed to read agent dashboard template: {str(e)}")

    def _extract_schema(self, model: Type[BaseModel], visited: Optional[Set[str]] = None) -> Dict[str, Any]:
        """Extract and inline schema for a model."""
        if visited is None:
            visited = set()

        schema = model.model_json_schema()
        self._inline_references(schema, visited)
        return schema

    def _inline_references(self, schema: Dict[str, Any], visited: Set[str]) -> None:
        """Recursively inline schema references."""
        if not isinstance(schema, dict):
            return

        if "$ref" in schema:
            ref_name = schema["$ref"].split("/")[-1]
            if ref_name in visited:
                return

            visited.add(ref_name)
            if ref_name in self._schema_definitions:
                ref_model = self._schema_definitions[ref_name]
                ref_schema = ref_model.model_json_schema()
                self._inline_references(ref_schema, visited)
                schema.clear()
                schema.update(ref_schema)
            return

        for key, value in schema.items():
            if isinstance(value, dict):
                self._inline_references(value, visited)
            elif isinstance(value, list):
                for item in value:
                    if isinstance(item, dict):
                        self._inline_references(item, visited)

        if "$defs" in schema:
            del schema["$defs"]

    def _format_action_endpoint(self, info: ActionEndpointInfo) -> Dict[str, Any]:
        """Format action endpoint information for output."""
        return {
            "name": info.metadata.name,
            "slug": slugify(info.metadata.name),
            "actionType": info.metadata.action_type,
            "path": info.route_path,
            "method": "POST",
            "inputSchema": self._extract_schema(info.input_model),
            "outputSchema": self._extract_schema(info.output_model),
            "examples": info.examples or {"validRequests": []},
            "description": info.metadata.description
        }

    def generate_manifest(self) -> Dict[str, Any]:
        """Generate the complete agent manifest."""
        if not all([self._base_url, self._name, self._version, self._description]):
            raise ValueError("Registry not properly configured. Missing required attributes.")

        return {
            "name": self._name,
            "slug": slugify(self._name),
            "version": self._version,
            "type": "external",
            "description": self._description,
            "baseUrl": self._base_url,
            "metaInfo": {},
            "capabilities": [
                cap.model_dump(exclude_none=True) 
                for cap in self._capabilities
            ],
            "actions": [
                self._format_action_endpoint(info) 
                for info in self._action_endpoints.values()
            ]
        }

# Global registry instance
registry = AgentRegistry()

def configure_agent(
    base_url: str,
    name: str,
    version: str,
    description: str,
    capabilities: List[Capability],
    mount_path: Optional[str] = None,
    html_path: Optional[str] = None,
    json_path: Optional[str] = None,
    dashboard_template_path: Optional[Union[str, Path]] = None,
):
    """Decorator to configure the agent."""
    def decorator(app: FastAPI):
        registry.configure(
            base_url=base_url,
            name=name,
            version=version,
            description=description,
            capabilities=capabilities,
            mount_path=mount_path,
            html_path=html_path,
            json_path=json_path,
            dashboard_template_path=dashboard_template_path
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
    """Decorator to register an agent action."""
    def decorator(func: Callable) -> Callable:
        sig = inspect.signature(func)
        
        # Get input model from function parameters
        input_model = next(
            (param.annotation for param in sig.parameters.values() 
             if hasattr(param.annotation, 'model_json_schema')),
            None
        )
        
        # Get output model from return annotation
        return_annotation = func.__annotations__.get('return')
        output_model = (
            return_annotation.__args__[0] 
            if hasattr(return_annotation, '__origin__') 
            else return_annotation
        )

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
            route_path=""
        )

        func._endpoint_info = endpoint_info

        @wraps(func)
        async def wrapper(*args, **kwargs):
            return await func(*args, **kwargs)

        return wrapper
    return decorator

def setup_agent_routes(app: FastAPI) -> None:
    """Set up all agent routes including registry endpoints."""
    
    def register_routes(routes, prefix=""):
        for route in routes:
            if isinstance(getattr(route, "app", None), FastAPI):
                mounted_prefix = prefix + str(route.path).rstrip("/")
                register_routes(route.app.routes, mounted_prefix)
            elif hasattr(route, "endpoint") and hasattr(route.endpoint, "_endpoint_info"):
                route_path = prefix + str(route.path)
                registry.register_action_endpoint(route_path, route.endpoint._endpoint_info)

    register_routes(app.routes)

    # Set up agents.json endpoint
    @app.get("/agents.json")
    async def get_agents_manifest():
        """Return list of all registered agents."""
        return {"agents": registry._registered_agents}

    # Set up the JSON manifest endpoint with dynamic path
    json_path = registry._json_path
    if not json_path:
        agent_slug = slugify(registry._name)
        json_path = f"/agents/{agent_slug}.json"

    @app.get(json_path)
    async def get_agent_manifest():
        """Return the complete agent manifest."""
        return registry.generate_manifest()

    # Set up the HTML endpoint with dynamic path
    html_path = registry._html_path
    if not html_path:
        agent_slug = slugify(registry._name)
        html_path = f"/agents/{agent_slug}"

    @app.get(html_path, response_class=HTMLResponse)
    async def get_agent_dashboard():
        """Return the agent dashboard page."""
        html_content = registry.get_agent_dashboard()
        if not html_content:
            raise HTTPException(status_code=404, detail="Agent dashboard template not configured")
        return html_content