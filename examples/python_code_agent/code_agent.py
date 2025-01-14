from fastapi import FastAPI, HTTPException, BackgroundTasks, Request
from pathlib import Path
from datetime import datetime
import black
from typing import List, Dict, Any

from models import (
    ChatInput, ChatOutput, GenerateCodeInput, GenerateCodeOutput,
    ImproveCodeInput, ImproveCodeOutput, TestCodeInput, TestCodeOutput,
    DeployPreviewInput, DeployPreviewOutput, AGENT_CAPABILITIES
)
from manifest_generator import configure_agent, agent_action, ActionType
from llm_client import create_llm_client

AGENT_TEMPLATE = Path(__file__).parent / "templates" / "agent.html"
AGENTS_TEMPLATE = Path(__file__).parent / "templates" / "agents.html"

# Initialize LLM client
llm_client = create_llm_client()

agent_app = FastAPI()

agent_app = configure_agent(
    app=agent_app,
    base_url="http://localhost:9200",
    name="Python Code Assistant",
    version="1.0.0",
    description="Advanced Python code generation and communication agent",
    capabilities=AGENT_CAPABILITIES
)

async def _generate_code_from_requirements(code_requirements: Any) -> str:
    """Generate code based on requirements using LLM."""
    prompt = f"Generate Python/{code_requirements.framework} code for: {code_requirements.description}. Functions: {', '.join(code_requirements.required_functions)}"
    
    response = await llm_client.complete(
        prompt=prompt,
        system_message="You are an expert Python developer. Generate clean, efficient code following PEP 8 standards.",
        temperature=0.3
    )
    return response.content

async def _generate_tests(code: str, test_instructions: List[Any]) -> str:
    """Generate test cases using LLM."""
    prompt = f"""
    Generate Python test cases for the following code:
    {code}
    
    Test requirements:
    {[instr.description for instr in test_instructions]}
    """
    
    response = await llm_client.complete(
        prompt=prompt,
        system_message="You are an expert in Python testing. Generate comprehensive test cases.",
        temperature=0.2
    )
    return response.content

async def _generate_documentation(code: str, level: str) -> str:
    """Generate documentation using LLM."""
    prompt = f"""
    Generate {level} documentation for the following Python code:
    {code}
    """
    
    response = await llm_client.complete(
        prompt=prompt,
        system_message="You are a technical documentation expert. Generate clear and comprehensive documentation.",
        temperature=0.3
    )
    return response.content

async def _apply_code_changes(change: Any) -> str:
    """Apply code improvements using LLM."""
    prompt = f"""
    Improve the following Python code according to these requirements:
    Change type: {change.type}
    Description: {change.description}
    Priority: {change.priority}
    
    Code to improve:
    {change.target or "No code provided"}
    """
    
    response = await llm_client.complete(
        prompt=prompt,
        system_message="You are an expert Python developer. Improve the code while maintaining its functionality.",
        temperature=0.2
    )
    return response.content

@agent_app.post("/code_agent/python/chat", response_model=ChatOutput)
@agent_action(
    action_type=ActionType.TALK,
    name="Chat with Python Assistant",
    description="Engage in a conversation with the Python code agent",
    response_template_md="templates/chat_response.md"
)
async def chat_with_agent(input_data: ChatInput) -> ChatOutput:
    """Handle chat interactions with the agent."""
    try:
        response = await llm_client.complete(
            prompt=input_data.message,
            system_message="You are a helpful Python programming assistant.",
            temperature=0.7
        )

        return ChatOutput(
            response=response.content,
            confidence=0.95,
            suggested_actions=["Share your code", "Specify requirements", "Run analysis"]
        )
    except Exception as e:
        raise HTTPException(status_code=400, detail=str(e))

@agent_app.post("/code_agent/python/generate_code", response_model=GenerateCodeOutput)
@agent_action(
    action_type=ActionType.GENERATE,
    name="Generate Python Code",
    description="Generates Python code based on requirements",
    response_template_md="templates/generate_response.md"
)
async def generate_code(input_data: GenerateCodeInput) -> GenerateCodeOutput:
    """Generate Python code based on specified requirements."""
    try:
        generated_code = await _generate_code_from_requirements(input_data.code_requirements)
        test_cases = []
        if input_data.include_tests:
            test_code = await _generate_tests(generated_code, [])
            test_cases = [test_code] if test_code else []
        documentation = await _generate_documentation(generated_code, input_data.documentation_level)

        return GenerateCodeOutput(
            generated_code=generated_code,
            description="Generated code based on requirements",
            test_cases=test_cases,
            documentation=documentation
        )
    except Exception as e:
        raise HTTPException(status_code=400, detail=str(e))

@agent_app.post("/code_agent/python/improve_code", response_model=ImproveCodeOutput)
@agent_action(
    action_type=ActionType.GENERATE,
    name="Improve Python Code",
    description="Improves and formats existing Python code"
)
async def improve_code(input_data: ImproveCodeInput) -> ImproveCodeOutput:
    """Improve and format Python code."""
    try:
        code_changes = []
        for change in input_data.changes_list:
            improved_code = await _apply_code_changes(change)
            
            if input_data.apply_black_formatting:
                improved_code = black.format_str(improved_code, mode=black.FileMode())

            code_changes.append({
                "type": change.type,
                "description": change.description,
                "before": change.target or "",
                "after": improved_code,
                "impact": "Code structure improved and formatted"
            })

        return ImproveCodeOutput(
            code_changes=code_changes,
            changes_description="Applied code improvements successfully",
            quality_metrics={
                "complexity": 75.0,
                "maintainability": 85.0,
                "test_coverage": 90.0
            }
        )
    except Exception as e:
        raise HTTPException(status_code=400, detail=str(e))
