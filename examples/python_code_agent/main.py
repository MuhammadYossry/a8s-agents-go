from fastapi import FastAPI, HTTPException, BackgroundTasks
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel, Field
from typing import List, Optional, Dict, Any, Literal
from enum import Enum
import ast
import black
import pylint.lint
from pathlib import Path
from datetime import datetime

from models import (
    BaseModelCamel, GenerateCodeInput, GenerateCodeOutput,
    ImproveCodeInput, ImproveCodeOutput, TestCodeInput, TestCodeOutput,
    DeployPreviewInput, DeployPreviewOutput, CodeRequirement, StyleGuide,
    CodeChange, CodeChangeOutput, QualityMetrics, CodingStyle, CoverageStatus,
    TestType, ChatInput, ChatOutput, ChatMessage, TestInstruction,
    AGENT_CAPABILITIES
)

from manifest_generator import configure_agent, agent_action, setup_agent_routes, ActionType

app = FastAPI()
agent_app = FastAPI()

AGENT_TEMPLATE = Path(__file__).parent / "templates" / "agent.html"

# Declare your agent
@configure_agent(
    base_url="http://localhost:9200",
    name="Python Code Assistant",  # Human readable name
    version="1.0.0",
    description="Advanced Python code generation and communication agent",
    capabilities=AGENT_CAPABILITIES,
    dashboard_template_path="templates/agent.html",
)
@agent_app.post("/code_agent/python/chat", response_model=ChatOutput)
@agent_action(
    action_type=ActionType.TALK,
    name="Chat with Python Assistant",
    description="Engage in a conversation with the Python code agent",
    schema_definitions={
        "ChatMessage": ChatMessage
    },
    examples={
        "validRequests": [
            {
                "message": "Can you help me optimize my Python code?",
                "context": "Performance optimization",
                "history": []
            }
        ]
    }
)
async def chat_with_agent(input_data: ChatInput) -> ChatOutput:
    """Handle chat interactions with the agent."""
    try:
        # Here you would integrate with your actual LLM/chat implementation
        response = "I'd be happy to help you with your Python code. Could you share the code you'd like to optimize?"
        
        return ChatOutput(
            response=response,
            confidence=0.95,
            suggested_actions=[
                "Share your code",
                "Specify performance requirements",
                "Run code analysis"
            ]
        )
    except Exception as e:
        raise HTTPException(status_code=400, detail=str(e))

@agent_app.post("/code_agent/python/generate_code", response_model=GenerateCodeOutput)
@agent_action(
    action_type=ActionType.GENERATE,
    name="Generate Python Code",
    description="Generates Python code based on requirements",
    schema_definitions={
        "CodeRequirement": CodeRequirement,
        "StyleGuide": StyleGuide,
        "CodingStyle": CodingStyle
    },
    examples={
        "validRequests": [
            {
                "codeRequirements": {
                    "language": "Python",
                    "framework": "FastAPI",
                    "description": "Create a REST API endpoint",
                    "requirements": [
                        "FastAPI",
                        "RESTful API design",
                        "HTTP methods"
                    ],
                    "requiredFunctions": [
                        "create_endpoint",
                        "handle_request",
                        "validate_input"
                    ],
                    "testingRequirements": [
                        "test_endpoint_creation",
                        "test_request_handling",
                        "test_input_validation"
                    ],
                    "codingStyle": {
                        "patterns": ["REST API", "Clean Architecture"],
                        "conventions": ["PEP 8", "FastAPI best practices"]
                    }
                },
                "styleGuide": {
                    "formatting": "black",
                    "maxLineLength": 88
                },
                "includeTests": True,
                "documentationLevel": "detailed"
            }
        ]
    }
)


async def generate_code(input_data: GenerateCodeInput) -> GenerateCodeOutput:
    """Generate Python code based on specified requirements."""
    try:
        generated_code = _generate_code_from_requirements(input_data.code_requirements)
        test_cases = _generate_test_cases(generated_code) if input_data.include_tests else []
        documentation = _generate_documentation(generated_code, input_data.documentation_level)
        
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
    description="Improves and formats existing Python code",
    schema_definitions={
        "CodeChange": CodeChange,
        "CodeChangeOutput": CodeChangeOutput,
        "QualityMetrics": QualityMetrics
    },
    examples={
        "validRequests": [
            {
                "changesList": [{
                    "type": "refactor",
                    "description": "Improve function structure",
                    "target": "main.py",
                    "priority": "medium"
                }],
                "applyBlackFormatting": True,
                "runLinter": True
            }
        ]
    }
)
async def improve_code(input_data: ImproveCodeInput) -> ImproveCodeOutput:
    """Improve and format Python code."""
    try:
        code_changes = []
        for change in input_data.changes_list:
            improved_code = _apply_code_changes(change)
            
            if input_data.apply_black_formatting:
                improved_code = black.format_str(improved_code, mode=black.FileMode())
            
            code_changes.append(CodeChangeOutput(
                type=change.type,
                description=change.description,
                before=change.target or "",
                after=improved_code,
                impact="Code structure improved and formatted"
            ))

        metrics = QualityMetrics(
            complexity=75.0,
            maintainability=85.0,
            test_coverage=90.0
        )

        return ImproveCodeOutput(
            code_changes=code_changes,
            changes_description="Applied code improvements successfully",
            quality_metrics=metrics
        )
    except Exception as e:
        raise HTTPException(status_code=400, detail=str(e))

@agent_app.post("/code_agent/python/test_code", response_model=TestCodeOutput)
@agent_action(
    action_type=ActionType.GENERATE,
    name="Write tests",
    description="Generates and runs tests for Python code",
    schema_definitions={
        "TestInstruction": TestInstruction,
        "CoverageStatus": CoverageStatus
    },
    examples={
        "validRequests": [
            {
                "testType": "unit",
                "requirePassing": True,
                "testInstructions": [{
                    "description": "Test API endpoints",
                    "assertions": ["test_status_code", "test_response_format"],
                    "testType": "unit"
                }],
                "codeToTest": "def example(): return True",
                "minimumCoverage": 80.0
            }
        ]
    }
)
async def test_code(input_data: TestCodeInput) -> TestCodeOutput:
    """Generate and run tests for Python code."""
    try:
        generated_tests = _generate_tests(input_data.code_to_test, input_data.test_instructions)
        coverage = CoverageStatus(percentage=85.0, uncovered_lines=[10, 15])

        return TestCodeOutput(
            code_tests=generated_tests,
            tests_description="Generated test suite with complete coverage",
            coverage_status=coverage
        )
    except Exception as e:
        raise HTTPException(status_code=400, detail=str(e))

@agent_app.post("/code_agent/python/deploy_preview", response_model=DeployPreviewOutput)
@agent_action(
    action_type=ActionType.GENERATE,
    name="Deploy a Preview",
    description="Creates a preview deployment for code review",
    examples={
        "validRequests": [
            {
                "branchId": "feature-123",
                "isPrivate": True,
                "environmentVars": {
                    "DEBUG": "true",
                    "API_KEY": "preview-key"
                }
            }
        ]
    }
)
async def deploy_preview(
    input_data: DeployPreviewInput,
    background_tasks: BackgroundTasks
) -> DeployPreviewOutput:
    """Create a preview deployment for code review."""
    try:
        preview_url = f"https://preview.example.com/{input_data.branch_id}"
        http_auth = {"username": "preview", "password": "secret"} if input_data.is_private else None
        
        background_tasks.add_task(_cleanup_preview, input_data.branch_id)
        
        return DeployPreviewOutput(
            preview_url=preview_url,
            is_private=input_data.is_private,
            http_auth=http_auth,
            deployment_time=datetime.now()
        )
    except Exception as e:
        raise HTTPException(status_code=400, detail=str(e))


# Set all CORS enabled origins
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*", "http://localhost:8000"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# Mount the agent app
app.mount("/v1", agent_app)

# Set up the agents.json endpoint
setup_agent_routes(app)

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=9200)
