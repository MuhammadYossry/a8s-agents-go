from functools import wraps
from typing import List, Dict, Any, Optional, Type, Callable, Union
from pydantic import BaseModel, Field
from fastapi import FastAPI, HTTPException
from fastapi.responses import HTMLResponse
from jinja2 import Environment, FileSystemLoader
import inspect
from enum import Enum
from pathlib import Path
import re

## Workflow
class WorkflowStepType(str, Enum):
    START = "start"
    ACTION = "action"
    END = "end"

class WorkflowTransition(BaseModel):
    target: str
    condition: Optional[str] = None

class WorkflowStep(BaseModel):
    id: str
    type: WorkflowStepType
    action: Optional[str] = None
    transitions: List[WorkflowTransition] = Field(default_factory=list)

class WorkflowMetadata(BaseModel):
   """Workflow-specific metadata for actions."""
   workflow_id: str
   step_id: str

class Workflow(BaseModel):
    id: str
    name: str
    description: str
    steps: List[WorkflowStep]
    initial_step: str

class AgentWorkflow(BaseModel):
    """Enhanced agent capabilities with workflow support."""
    workflows: List[Workflow] = Field(default_factory=list)
    actions: Dict[str, Dict[str, Any]] = Field(default_factory=dict)

# Actions and manifest generations

def slugify(text: str) -> str:
    """Convert text to URL-safe slug."""
    text = text.lower()
    text = re.sub(r'[^\w\s-]', '', text)
    text = re.sub(r'[-\s]+', '-', text)
    return text.strip('-')

def get_action_context(endpoint_info, agent_slug: str, action_slug: str) -> dict:
    """Generate template context from endpoint info"""
    input_schema = endpoint_info.input_model.model_json_schema()
    output_schema = endpoint_info.output_model.model_json_schema()
    
    context = {
        "action": {
            "name": endpoint_info.metadata.name,
            "description": endpoint_info.metadata.description,
            "actionType": endpoint_info.metadata.action_type.value,
            "path": endpoint_info.route_path,
            "method": "POST",
            "inputSchema": input_schema,
            "outputSchema": output_schema,
            "slug": action_slug,
            "isMDResponseEnabled": endpoint_info.metadata.response_template_md is not None
        },
        "agent_slug": agent_slug,
        "action_slug": action_slug
    }

    if endpoint_info.examples:
        context["action"]["examples"] = endpoint_info.examples
        
    if endpoint_info.metadata.response_template_md:
        template_path = Path(endpoint_info.metadata.response_template_md)
        if template_path.exists():
            context["action"]["responseTemplateMD"] = template_path.read_text()

    return context

class Capability(BaseModel):
    """Simplified capability definition."""
    skill_path: List[str]
    metadata: Dict[str, Any]

class ActionType(str, Enum):
    """Types of actions an agent can perform."""
    TALK = "talk"
    GENERATE = "generate"
    QUESTION = "question"

class ActionMetadata(BaseModel):
    action_type: ActionType
    name: str
    description: str
    response_template_md: Optional[str] = None
    workflow_id: Optional[str] = None  # Reference to workflow if part of one
    step_id: Optional[str] = None  # Reference to step in workflow

class ActionEndpointInfo(BaseModel):
    """Information about an action endpoint."""
    metadata: ActionMetadata
    input_model: Type[BaseModel]
    output_model: Type[BaseModel]
    schema_definitions: Optional[Dict[str, Type[BaseModel]]] = None
    examples: Optional[Dict[str, List[Dict[str, Any]]]] = None
    route_path: str

class ActionContext(BaseModel):
    name: str
    description: str 
    action_type: str
    agent_slug: str
    action_slug: str
    input_schema: Dict[str, Any]
    output_schema: Dict[str, Any]
    route_path: str
    response_template_md: str = None
    examples: Dict[str, Any] = None


import logging

logging.basicConfig(level=logging.DEBUG)
logger = logging.getLogger(__name__)

class AgentRegistry:
    def __init__(self, base_url: str, name: str, version: str, description: str, capabilities: List[Capability], workflows: List[Workflow]):
        logger.debug(f"Initializing AgentRegistry for {name}")
        self.base_url = base_url.rstrip('/')
        self.name = name
        self.slug = slugify(name)
        self.version = version
        self.description = description
        self.capabilities = capabilities
        self.action_endpoints: Dict[str, ActionEndpointInfo] = {}
        self.schema_definitions: Dict[str, Dict[str, Any]] = {}
        self.workflows = workflows or []
        logger.debug(f"Registry initialized with slug: {self.slug}")

    def _format_action_endpoint(self, info: ActionEndpointInfo) -> Dict[str, Any]:
        endpoint_data = {
            "name": info.metadata.name,
            "slug": slugify(info.metadata.name),
            "actionType": info.metadata.action_type,
            "path": info.route_path,
            "method": "POST",
            "inputSchema": self._extract_schema(info.input_model),
            "outputSchema": self._extract_schema(info.output_model),
            "examples": info.examples or {"validRequests": []},
            "description": info.metadata.description,
            "isMDResponseEnabled": info.metadata.response_template_md is not None
        }
        if info.metadata.response_template_md:
            try:
                template_path = Path(__file__).parent / info.metadata.response_template_md
                if template_path.exists():
                    template_content = template_path.read_text()
                    endpoint_data["responseTemplateMD"] = template_content
                    info.metadata.response_template_content = template_content
            except Exception as e:
                logger.warning(f"Failed to read template {info.metadata.response_template_md}: {e}")
        return endpoint_data

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
        if self.workflows:
            manifest["workflows"] = [
                {
                    "id": w.id,
                    "name": w.name,
                    "description": w.description,
                    "steps": [step.model_dump() for step in w.steps],
                    "initial_step": w.initial_step
                }
                for w in self.workflows
            ]
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
    workflows: List[Workflow] = None,
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
    registry = AgentRegistry(base_url, name, version, description, capabilities, workflows)
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
    response_template_md: Optional[str] = None,
    schema_definitions: Optional[Dict[str, Type[BaseModel]]] = None,
    examples: Optional[Dict[str, List[Dict[str, Any]]]] = None,
    workflow_id: Optional[str] = None,
    step_id: Optional[str] = None
) -> Callable:
    def decorator(func: Callable) -> Callable:
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
        workflow_meta = None
        if workflow_id and step_id:
            workflow_meta = WorkflowMetadata(
                workflow_id=workflow_id,
                step_id=step_id
            )
        endpoint_info = ActionEndpointInfo(
            metadata=ActionMetadata(
                action_type=action_type,
                name=name,
                description=description,
                response_template_md=response_template_md,
                workflow=workflow_meta
            ),
            input_model=input_model,
            output_model=output_model,
            schema_definitions=schema_definitions,
            examples=examples,
            route_path=""
        )

        @wraps(func)
        async def wrapper(*args, **kwargs):
            markdown = kwargs.pop('markdown', False)
            result = await func(*args, **kwargs)
            
            if markdown and endpoint_info.metadata.response_template_md:
                if not isinstance(result, dict):
                    result = result.model_dump()
                
                template_path = Path(endpoint_info.metadata.response_template_md)
                if template_path.exists():
                    template_content = template_path.read_text()
                    from jinja2 import Template
                    rendered = Template(template_content).render(**result)
                    return Response(content=rendered, media_type="text/markdown")
            
            return result
            
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
    for agent_slug, registry in agent_registries.items():
        # Add manifest endpoint
        @app.get(f"/agents/{agent_slug}.json")
        async def get_agent_manifest(reg=registry):
            return reg.generate_manifest()

        # Add dashboard endpoint
        @app.get(f"/agents/{agent_slug}", response_class=HTMLResponse)
        async def get_agent_dashboard(reg=registry):
            try:
                template_path = templates_dir / "agent.html"
                return template_path.read_text()
            except Exception as e:
                raise HTTPException(status_code=404, detail="Agent dashboard template not found")


        for route_path, endpoint_info in registry.action_endpoints.items():
            action_slug = slugify(endpoint_info.metadata.name)
            
            @app.get(f"/agents/{agent_slug}/actions/{action_slug}", response_class=HTMLResponse)
            async def get_agent_action_page(
                agent_slug=agent_slug, 
                action_slug=action_slug,
                reg=registry,
                endpoint_info=endpoint_info
            ):
                try:
                    templates_dir = Path(__file__).parent / "templates"
                    env = Environment(
                        loader=FileSystemLoader(templates_dir),
                        autoescape=True
                    )
                    
                    template = env.get_template("agent_action.html")
                    context = get_action_context(endpoint_info, agent_slug, action_slug)
                    
                    return template.render(**context)
                    
                except Exception as e:
                    raise HTTPException(
                        status_code=404,
                        detail=f"Error rendering action page: {str(e)}"
                    )