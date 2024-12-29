from typing import Dict, Any, List, Optional
from manifest_generator import registry, AgentEndpointRegistry
from fastapi import FastAPI, APIRouter
from fastapi.responses import Response
import json
from textwrap import indent, dedent

class ASDLGenerator:
    """Generates Agent Service Definition Language (ASDL) v1.0 documentation."""
    
    def __init__(self, registry: AgentEndpointRegistry):
        self.registry = registry
        self.indent_str = "    "

    def _indent(self, text: str, level: int = 1) -> str:
        """Indent text by specified level."""
        return indent(text, self.indent_str * level)

    def _format_list_values(self, values: List[str]) -> str:
        """Format a list of values for ASDL."""
        return ', '.join(f'"{v}"' for v in values)

    def _format_metadata_capabilities(self, metadata: Dict[str, Any]) -> List[str]:
        """Format metadata into capability statements."""
        capabilities = []
        for key, value in metadata.items():
            if isinstance(value, list):
                cap_type = key[:-1] if key.endswith('s') else key
                capabilities.append(f'{cap_type} {self._format_list_values(value)}')
        return capabilities

    def _format_cognitive_abilities(self, capabilities: List[Dict[str, Any]]) -> str:
        """Format capabilities as cognitive abilities blocks."""
        if not capabilities:
            return ""
            
        # Group capabilities by domain
        domains = {}
        for cap in capabilities:
            path = cap["skillPath"]
            domain = path[0]
            if domain not in domains:
                domains[domain] = []
            domains[domain].append(cap)
        
        asdl = "cognitive_abilities {\n"
        
        for domain, caps in domains.items():
            # Find domain level capability
            domain_cap = next((c for c in caps if len(c["skillPath"]) == 1), None)
            domain_meta = domain_cap["metadata"] if domain_cap else {}
            
            asdl += self._indent(f'knowledge_domain {domain} {{\n')
            for key, value in domain_meta.items():
                if isinstance(value, str):
                    asdl += self._indent(f'proficiency_{key}: {value}\n', 2)
            
            # Process specialties
            specialties = [c for c in caps if len(c["skillPath"]) > 1]
            for specialty in specialties:
                path = specialty["skillPath"][1:]
                meta = specialty["metadata"]
                
                if len(path) == 1:  # Specialty level
                    asdl += self._indent(f'specialization {".".join(path)} {{\n', 2)
                    for key, value in meta.items():
                        if isinstance(value, str):
                            asdl += self._indent(f'proficiency_{key}: {value}\n', 3)
                    
                    capabilities = self._format_metadata_capabilities(meta)
                    if capabilities:
                        asdl += self._indent('capabilities: [\n', 3)
                        asdl += self._indent('\n'.join(capabilities) + '\n', 4)
                        asdl += self._indent(']\n', 3)
                        
                elif len(path) == 2:  # Skill level
                    asdl += self._indent(f'skill {path[1]} {{\n', 3)
                    for key, value in meta.items():
                        if isinstance(value, (str, list)):
                            formatted_value = json.dumps(value) if isinstance(value, list) else value
                            asdl += self._indent(f'{key}: {formatted_value}\n', 4)
                    asdl += self._indent('}\n', 3)
            
                if len(path) == 1:
                    asdl += self._indent('}\n', 2)
            
            asdl += self._indent('}\n')
            
        return asdl + '}'

    def _format_schema_type(self, schema: Dict[str, Any]) -> str:
        """Convert JSON schema type to ASDL type definition."""
        if "const" in schema:
            return f'fixed({json.dumps(schema["const"])})'
        elif "enum" in schema:
            return f'oneof({", ".join(json.dumps(v) for v in schema["enum"])})'
        elif schema.get("type") == "array":
            item_type = self._format_schema_type(schema["items"])
            return f'collection<{item_type}>'
        elif schema.get("type") == "number":
            constraints = []
            if "minimum" in schema:
                constraints.append(str(schema["minimum"]))
            if "maximum" in schema:
                constraints.append(str(schema["maximum"]))
            if constraints:
                return f'numeric({".." if len(constraints) > 1 else ""}{".".join(constraints)})'
            return "numeric"
        elif schema.get("type") == "boolean":
            return "logical"
        return "content"

    def _format_property(self, name: str, schema: Dict[str, Any], required: bool) -> str:
        """Format a property definition."""
        prop_type = self._format_schema_type(schema)
        if "default" in schema:
            return f'{name}: {prop_type} = {json.dumps(schema["default"])}'
        return f'{"required " if required else ""}{name}: {prop_type}'

    def _format_schema(self, schema: Dict[str, Any], indent_level: int = 0) -> str:
        """Format complete schema definition."""
        asdl = ""
        required = schema.get("required", [])
        
        for prop_name, prop_schema in schema.get("properties", {}).items():
            is_required = prop_name in required
            
            if prop_schema.get("type") == "object":
                asdl += self._indent(f'{"required " if is_required else ""}{"optional " if not is_required else ""}{prop_name} {{\n', indent_level)
                asdl += self._format_schema(prop_schema, indent_level + 1)
                asdl += self._indent('}\n', indent_level)
            else:
                asdl += self._indent(self._format_property(prop_name, prop_schema, is_required) + '\n', indent_level)
        
        return asdl

    def _format_interaction(self, name: str, endpoint: Dict[str, Any]) -> str:
        """Format an endpoint as an interaction definition."""
        asdl = f'interaction {name} {{\n'
        asdl += self._indent(f'endpoint: "{endpoint["path"]}"\n')
        asdl += self._indent(f'protocol: {endpoint["method"]}\n\n')
        
        # Input schema
        if "inputSchema" in endpoint:
            asdl += self._indent('expects {\n')
            asdl += self._format_schema(endpoint["inputSchema"], 2)
            asdl += self._indent('}\n\n')
        
        # Output schema
        if "outputSchema" in endpoint:
            asdl += self._indent('provides {\n')
            asdl += self._format_schema(endpoint["outputSchema"], 2)
            asdl += self._indent('}\n\n')
        
        # Examples
        if "examples" in endpoint and endpoint["examples"].get("validRequests"):
            example = endpoint["examples"]["validRequests"][0]
            asdl += self._indent('behavior_example {\n')
            asdl += self._indent('input ' + json.dumps(example, indent=4).replace('\n', '\n' + self.indent_str * 2))
            asdl += '\n' + self._indent('}\n\n')
        
        # Error responses
        if "errorResponses" in endpoint:
            asdl += self._indent('error_handlers {\n')
            for code, msg in endpoint["errorResponses"].items():
                asdl += self._indent(f'{code}: "{msg}"\n', 2)
            asdl += self._indent('}\n')
        
        return asdl + '}'

    def generate_asdl(self) -> str:
        """Generate complete ASDL documentation."""
        config = self.registry.generate_config()
        if not config.get("agents"):
            return "# No agents configured"
        
        agent = config["agents"][0]  # Assume first agent
        agent_type = agent.get("type", "AI.AgentDefinition")
        
        asdl = f'''agent {agent["id"]} {{
    category: {agent_type}
    version: "1.0"
    base_url: "{agent["baseURL"]}"
    purpose: """{agent["description"]}"""
    
    # Cognitive abilities and knowledge domains
    {self._format_cognitive_abilities(agent["capabilities"])}
    
    # Interaction protocols
    behaviors {{
'''
        
        # Add interactions
        for action in agent["actions"]:
            asdl += self._indent(self._format_interaction(action["name"], action)) + '\n'
        
        asdl += "    }\n}"
        
        return asdl

def extend_app_with_asdl(app: FastAPI) -> None:
    """Set up the agents.asdl endpoint."""
    asdl_generator = ASDLGenerator(registry)
    router = APIRouter()
    
    @router.get("/agents.asdl")
    async def get_agent_asdl():
        return Response(
            content=asdl_generator.generate_asdl(),
            media_type="text/plain"
        )
    
    app.include_router(router)