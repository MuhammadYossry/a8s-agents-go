from functools import wraps
from typing import List, Dict, Any, Optional, Type, Callable, Union
from pydantic import BaseModel
from fastapi import FastAPI, HTTPException
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

class Capability(BaseModel):
    """Simplified capability definition."""
    skill_path: List[str]
    metadata: Dict[str, Any]

class ActionType(str, Enum):
    """Types of actions an agent can perform."""
    TALK = "talk"
    GENERATE = "generate"

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

import logging

logging.basicConfig(level=logging.DEBUG)
logger = logging.getLogger(__name__)

class AgentRegistry:
    def __init__(self, base_url: str, name: str, version: str, description: str, capabilities: List[Capability]):
        logger.debug(f"Initializing AgentRegistry for {name}")
        self.base_url = base_url.rstrip('/')
        self.name = name
        self.slug = slugify(name)
        self.version = version
        self.description = description
        self.capabilities = capabilities
        self.action_endpoints: Dict[str, ActionEndpointInfo] = {}
        self.schema_definitions: Dict[str, Dict[str, Any]] = {}
        logger.debug(f"Registry initialized with slug: {self.slug}")

    def _format_action_endpoint(self, info: ActionEndpointInfo) -> Dict[str, Any]:
        """Format endpoint info for output."""
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

    def _extract_schema(self, model: Type[BaseModel]) -> Dict[str, Any]:
        """Extract schema from model and process all references."""
        schema = model.model_json_schema()
        if '$defs' in schema:
            self.schema_definitions.update(schema['$defs'])
            del schema['$defs']
        return self._inline_references(schema)

    def _inline_references(self, schema: Dict[str, Any]) -> Dict[str, Any]:
        """Recursively inline all references in schema."""
        if not isinstance(schema, dict):
            return schema

        if '$ref' in schema:
            ref_name = schema['$ref'].split('/')[-1]
            if ref_name in self.schema_definitions:
                inlined = self.schema_definitions[ref_name].copy()
                return self._inline_references(inlined)
            return schema

        return {
            key: (
                [self._inline_references(item) for item in value]
                if isinstance(value, list)
                else (
                    self._inline_references(value)
                    if isinstance(value, dict)
                    else value
                )
            )
            for key, value in schema.items()
            if key != '$defs'
        }

    def register_action_endpoint(self, path: str, endpoint_info: ActionEndpointInfo) -> None:
        """Register an action endpoint with debug logging."""
        logger.debug(f"Registering action endpoint for path: {path}")
        logger.debug(f"Endpoint info: {endpoint_info.metadata.name}")
        self.action_endpoints[path] = endpoint_info
        if endpoint_info.schema_definitions:
            for key, model in endpoint_info.schema_definitions.items():
                logger.debug(f"Registering schema definition: {key}")
                self.schema_definitions[key] = model.model_json_schema()
        logger.debug(f"Total registered endpoints: {len(self.action_endpoints)}")

    def generate_manifest(self) -> Dict[str, Any]:
        """Generate manifest with debug logging."""
        logger.debug(f"Generating manifest for {self.name}")
        logger.debug(f"Number of registered actions: {len(self.action_endpoints)}")
        
        actions = [self._format_action_endpoint(info) for info in self.action_endpoints.values()]
        logger.debug(f"Formatted {len(actions)} actions")
        
        manifest = {
            "name": self.name,
            "slug": self.slug,
            "version": self.version,
            "type": "external",
            "description": self.description,
            "baseUrl": self.base_url,
            "metaInfo": {},
            "capabilities": [cap.model_dump(exclude_none=True) for cap in self.capabilities],
            "actions": actions
        }
        logger.debug(f"Generated manifest with {len(manifest['actions'])} actions")
        return manifest

# Global registries storage
agent_registries: Dict[str, AgentRegistry] = {}

def configure_agent(
    app: FastAPI,
    base_url: str,
    name: str,
    version: str,
    description: str,
    capabilities: List[Capability],
) -> FastAPI:
    """Configure a FastAPI app as an agent.
    Args:
        app: The FastAPI application to configure
        base_url: Base URL for the agent
        name: Name of the agent
        version: Version string
        description: Agent description
        capabilities: List of agent capabilities
    Returns:
        The configured FastAPI app
    """
    logger.debug(f"Configuring agent: {name}")
    # Create registry
    registry = AgentRegistry(base_url, name, version, description, capabilities)
    agent_registries[registry.slug] = registry

    # Add registry to app state
    if not hasattr(app, 'state'):
        setattr(app, 'state', type('State', (), {}))
    app.state.agent_registry = registry
    logger.debug(f"Created registry for {name} with slug {registry.slug}")
    return app

def agent_action(
    action_type: ActionType,
    name: str,
    description: str,
    schema_definitions: Optional[Dict[str, Type[BaseModel]]] = None,
    examples: Optional[Dict[str, List[Dict[str, Any]]]] = None
) -> Callable:
    """Enhanced agent_action decorator with better debugging"""
    def decorator(func: Callable) -> Callable:
        logger.debug(f"Decorating function {func.__name__} as agent action: {name}")
        # Get input/output models from function signature
        sig = inspect.signature(func)
        input_model = next(
            (param.annotation for param in sig.parameters.values() 
             if hasattr(param.annotation, 'model_json_schema')),
            None
        )
        output_model = (
            func.__annotations__.get('return').__args__[0] 
            if hasattr(func.__annotations__.get('return', None), '__origin__')
            else func.__annotations__.get('return')
        )
        if not input_model or not output_model:
            logger.error(f"Missing type annotations for {func.__name__}")
            raise ValueError(
                f"Both input and output models must be Pydantic models for {func.__name__}"
            )
        logger.debug(f"Action {name} input model: {input_model.__name__}")
        logger.debug(f"Action {name} output model: {output_model.__name__}")

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
            route_path=""  # Will be set during route registration
        )

        # Store the endpoint info in the function
        func._endpoint_info = endpoint_info
        logger.debug(f"Attached endpoint info to {func.__name__}")

        @wraps(func)
        async def wrapper(*args, **kwargs):
            logger.debug(f"Executing agent action: {name}")
            return await func(*args, **kwargs)

        wrapper._endpoint_info = endpoint_info
        return wrapper
    return decorator

def setup_agent_routes(app: FastAPI) -> None:
    """Enhanced setup_agent_routes with better debugging"""
    logger.debug("Setting up agent routes")
    templates_dir = Path(__file__).parent / "templates"

    def register_routes(routes, prefix="", parent_app=None, depth=0):
        indent = "  " * depth
        logger.debug(f"{indent}Registering routes with prefix: {prefix}")

        for route in routes:
            if isinstance(getattr(route, "app", None), FastAPI):
                mounted_app = route.app
                mounted_prefix = prefix + str(route.path).rstrip("/")
                logger.debug(f"{indent}Found mounted app at {mounted_prefix}")
                # Check if app has an agent registry
                if hasattr(mounted_app, "state") and hasattr(mounted_app.state, "agent_registry"):
                    registry = mounted_app.state.agent_registry
                    logger.debug(f"{indent}Found registry for {registry.name} on mounted app")
                    # Register routes from the mounted app
                    for mounted_route in mounted_app.routes:
                        if hasattr(mounted_route, "endpoint"):
                            logger.debug(f"{indent}Checking route: {mounted_route.path}")
                            endpoint = mounted_route.endpoint
                            if hasattr(endpoint, "_endpoint_info"):
                                endpoint_info = endpoint._endpoint_info
                                full_path = f"{mounted_prefix}{str(mounted_route.path)}"
                                logger.debug(f"{indent}Registering endpoint: {full_path}")
                                # Update the route path in endpoint info
                                endpoint_info.route_path = full_path
                                # Register with the correct registry
                                registry.register_action_endpoint(full_path, endpoint_info)
                                logger.debug(f"{indent}Registered {full_path} with {registry.name}")
                # Recursively process nested routes
                register_routes(mounted_app.routes, mounted_prefix, mounted_app, depth + 1)
    
    # Clear existing registrations
    logger.debug("Clearing existing registrations")
    for reg in agent_registries.values():
        reg.action_endpoints.clear()
    # Register all routes
    register_routes(app.routes)
    # Set up agents.json endpoint
    @app.get("/agents.json")
    async def get_agents_manifest():
        """Return list of all registered agents."""
        agents = []
        for registry in agent_registries.values():
            agents.append({
                "name": registry.name,
                "slug": registry.slug,
                "version": registry.version,
                "manifestUrl": f"{registry.base_url}/agents/{registry.slug}.json",
                "dashboardUrl": f"{registry.base_url}/agents/{registry.slug}"
            })
        return {"agents": agents}

    # Set up agents dashboard
    @app.get("/agents", response_class=HTMLResponse)
    async def get_agents_dashboard():
        """Return HTML page listing all agents."""
        try:
            template_path = templates_dir / "agents.html"
            return template_path.read_text()
        except Exception as e:
            raise HTTPException(status_code=404, detail="Agents dashboard template not found")

    # Set up individual agent endpoints
    for slug, registry in agent_registries.items():
        # Add manifest endpoint
        @app.get(f"/agents/{slug}.json")
        async def get_agent_manifest(reg=registry):
            return reg.generate_manifest()

        # Add dashboard endpoint
        @app.get(f"/agents/{slug}", response_class=HTMLResponse)
        async def get_agent_dashboard(reg=registry):
            try:
                template_path = templates_dir / "agent.html"
                return template_path.read_text()
            except Exception as e:
                raise HTTPException(status_code=404, detail="Agent dashboard template not found")