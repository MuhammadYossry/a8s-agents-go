from typing import Dict, Any, List, Optional
from manifest_generator import registry, AgentEndpointRegistry
from fastapi import FastAPI
from fastapi.responses import PlainTextResponse, Response
from fastapi import FastAPI, APIRouter
import json
from datetime import datetime
from tabulate import tabulate

class MarkdownGenerator:
    """Enhanced Markdown documentation generator with detailed schema information."""
    
    def __init__(self, registry: AgentEndpointRegistry):
        self.registry = registry

    def _format_property_table(self, properties: Dict[str, Any], required_fields: List[str]) -> str:
        """Format properties as a markdown table with detailed information."""
        headers = ["Field", "Type", "Required", "Description", "Default", "Constraints"]
        rows = []
        
        for prop_name, prop_details in properties.items():
            prop_type = self._get_property_type(prop_details)
            is_required = "✓" if prop_name in required_fields else ""
            description = prop_details.get("description", "")
            default = self._get_default_value(prop_details)
            constraints = self._get_constraints(prop_details)
            
            rows.append([
                f"`{prop_name}`",
                f"`{prop_type}`",
                is_required,
                description,
                default,
                constraints
            ])
        
        return tabulate(rows, headers, tablefmt="pipe")

    def _get_property_type(self, prop_details: Dict[str, Any]) -> str:
        """Get detailed type information including enums."""
        base_type = prop_details.get("type", "any")
        if "enum" in prop_details:
            return f"{base_type} (enum: {', '.join(map(str, prop_details['enum']))})"
        if "const" in prop_details:
            return f"{base_type} (const: {prop_details['const']})"
        if base_type == "array":
            items_type = self._get_property_type(prop_details.get("items", {}))
            return f"array of {items_type}"
        return base_type

    def _get_default_value(self, prop_details: Dict[str, Any]) -> str:
        """Get default value if present."""
        if "default" in prop_details:
            return f"`{json.dumps(prop_details['default'])}`"
        return "-"

    def _get_constraints(self, prop_details: Dict[str, Any]) -> str:
        """Extract and format all constraints."""
        constraints = []
        
        # Number constraints
        for key in ["minimum", "maximum", "exclusiveMinimum", "exclusiveMaximum"]:
            if key in prop_details:
                constraints.append(f"{key}: {prop_details[key]}")
        
        # String constraints
        for key in ["minLength", "maxLength", "pattern"]:
            if key in prop_details:
                constraints.append(f"{key}: {prop_details[key]}")
        
        # Array constraints
        for key in ["minItems", "maxItems", "uniqueItems"]:
            if key in prop_details:
                constraints.append(f"{key}: {prop_details[key]}")
        
        return ", ".join(constraints) if constraints else "-"

    def _format_examples(self, examples: Dict[str, List[Dict]], title: str = "Examples") -> str:
        """Format examples with descriptive text."""
        md = f"\n#### {title}\n\n"
        
        if "validRequests" in examples:
            md += "**Valid Requests:**\n\n"
            for i, example in enumerate(examples["validRequests"], 1):
                md += f"Example {i}:\n```json\n{json.dumps(example, indent=2)}\n```\n\n"
        
        if "invalidRequests" in examples:
            md += "**Invalid Requests (for reference):**\n\n"
            for i, example in enumerate(examples["invalidRequests"], 1):
                md += f"Example {i}:\n```json\n{json.dumps(example, indent=2)}\n```\n\n"
        
        return md

    def _format_capability(self, capability: Dict[str, Any]) -> str:
        """Format a capability with detailed metadata."""
        skill_path = " → ".join(capability["skillPath"])
        md = f"### {skill_path}\n\n"
        
        metadata = capability.get("metadata", {})
        if metadata:
            headers = ["Property", "Value"]
            rows = []
            for key, value in metadata.items():
                if value:
                    formatted_value = value if isinstance(value, str) else ", ".join(value)
                    rows.append([f"**{key}**", formatted_value])
            
            md += tabulate(rows, headers, tablefmt="pipe") + "\n\n"
        
        return md

    def _format_error_responses(self, schema: Dict[str, Any]) -> str:
        """Format possible error responses."""
        md = "\n#### Error Responses\n\n"
        
        common_errors = {
            "400": "Bad Request - Invalid input parameters",
            "401": "Unauthorized - Authentication required",
            "403": "Forbidden - Insufficient permissions",
            "404": "Not Found - Resource not found",
            "422": "Unprocessable Entity - Validation error",
            "500": "Internal Server Error - Server-side error occurred"
        }
        
        headers = ["Status Code", "Description", "Example Response"]
        rows = []
        
        for code, description in common_errors.items():
            example = {
                "error": {
                    "code": code,
                    "message": description,
                    "details": "Additional error context would appear here"
                }
            }
            rows.append([
                code,
                description,
                f"```json\n{json.dumps(example, indent=2)}\n```"
            ])
        
        return md + tabulate(rows, headers, tablefmt="pipe") + "\n"

    def _format_endpoint(self, endpoint: Dict[str, Any]) -> str:
        """Format an endpoint with comprehensive documentation."""
        md = f"### {endpoint['name']}\n\n"
        md += f"**Endpoint:** `{endpoint['method']} {endpoint['path']}`\n\n"
        
        # Input Schema
        if "inputSchema" in endpoint:
            md += "#### Input Schema\n\n"
            input_schema = endpoint["inputSchema"]
            
            if "description" in input_schema:
                md += f"{input_schema['description']}\n\n"
            
            if "properties" in input_schema:
                md += "**Properties:**\n\n"
                md += self._format_property_table(
                    input_schema["properties"],
                    input_schema.get("required", [])
                )
                md += "\n\n"
            
            # Parameter Interactions
            if "propertyDependencies" in input_schema:
                md += "**Parameter Dependencies:**\n\n"
                for prop, deps in input_schema["propertyDependencies"].items():
                    md += f"- When `{prop}` is present:\n"
                    for dep in deps:
                        md += f"  - `{dep}` is required\n"
                md += "\n"
        
        # Output Schema
        if "outputSchema" in endpoint:
            md += "#### Output Schema\n\n"
            output_schema = endpoint["outputSchema"]
            
            if "description" in output_schema:
                md += f"{output_schema['description']}\n\n"
            
            if "properties" in output_schema:
                md += "**Properties:**\n\n"
                md += self._format_property_table(
                    output_schema["properties"],
                    output_schema.get("required", [])
                )
                md += "\n\n"
        
        # Examples
        if "examples" in endpoint:
            md += self._format_examples(endpoint["examples"])
        
        # Error Responses
        md += self._format_error_responses(endpoint.get("errorResponses", {}))
        
        return md

    def generate_markdown(self) -> str:
        """Generate complete Markdown documentation."""
        config = self.registry.generate_config()
        if not config.get("agents"):
            return "No agents configured."
        
        md = "# AI Agents Service Documentation\n\n"
        md += "Welcome to our AI Agents service documentation. This service hosts several AI agents, each providing specific capabilities through well-documented endpoints. Below you'll find detailed information about each agent, their capabilities, and how to interact with them.\n\n"
        
        for agent in config["agents"]:
            md += f"## {agent['id']}\n\n"
            md += f"**Description:** {agent.get('description', 'No description provided.')}\n\n"
            md += f"**Base URL:** `{agent.get('baseURL', '')}`\n\n"
            
            # Capabilities
            if agent.get("capabilities"):
                md += "## Capabilities\n\n"
                md += "The following sections detail the specific capabilities of this agent:\n\n"
                for capability in agent["capabilities"]:
                    md += self._format_capability(capability)
            
            # Actions/Endpoints
            if agent.get("actions"):
                md += "## Available Endpoints\n\n"
                md += "This section describes all available endpoints for interacting with the agent:\n\n"
                for action in agent["actions"]:
                    md += self._format_endpoint(action)
        
        return md

def extend_app_with_markdown(app: FastAPI) -> None:
    """Set up the agents.md endpoint alongside agents.json."""
    markdown_generator = MarkdownGenerator(registry)
    router = APIRouter()
    
    @app.get("/agents.md", response_class=PlainTextResponse)
    async def get_agent_markdown():
        content = markdown_generator.generate_markdown()
        return Response(
            content=content,
            media_type="text/markdown"
        )

    app.include_router(router)

# Example usage:
# In your main.py, after setup_agent_routes(app), add:
# extend_app_with_markdown(app)