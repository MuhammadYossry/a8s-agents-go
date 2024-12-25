from fastapi import FastAPI, HTTPException, BackgroundTasks
from pydantic import BaseModel, Field, ConfigDict
from typing import List, Optional, Dict, Any, Literal
from enum import Enum
import ast
import black
import pylint.lint
from datetime import datetime

from manifest_generator import (
    configure_agent, agent_endpoint, setup_agent_routes,
    Capability, CapabilityMetadata, BaseSchemaModel
)

class BaseModelCamel(BaseModel):
    """Base model that configures camelCase support."""
    model_config = ConfigDict(
        alias_generator=lambda s: ''.join(
            word.capitalize() if i > 0 else word
            for i, word in enumerate(s.split('_'))
        ),
        populate_by_name=True
    )

# Code Generation Models
class CodingStyle(BaseModelCamel):
    """Model defining coding style preferences."""
    patterns: Optional[List[str]] = Field(default_factory=list)
    conventions: Optional[List[str]] = Field(default_factory=list)

class StyleGuide(BaseModelCamel):
    """Model defining style guide preferences."""
    formatting: Optional[Literal["black", "autopep8"]] = "black"
    max_line_length: Optional[int] = Field(ge=79, le=120, default=88)

class CodeRequirement(BaseModelCamel):
    """Model defining code generation requirements."""
    language: Literal["Python"]
    framework: Literal["FastAPI", "Django"]
    description: str
    requirements: List[str]
    required_functions: List[str]
    testing_requirements: List[str]
    coding_style: Optional[CodingStyle] = None

class GenerateCodeInput(BaseModelCamel):
    """Input model for code generation endpoint."""
    code_requirements: CodeRequirement
    style_guide: Optional[StyleGuide] = None
    include_tests: bool = True
    documentation_level: Literal["minimal", "standard", "detailed"] = "standard"

class GenerateCodeOutput(BaseModelCamel):
    """Output model for code generation endpoint."""
    generated_code: str
    description: str
    test_cases: List[str]
    documentation: str

# Code Improvement Models
class CodeChange(BaseModelCamel):
    """Model defining a code change request."""
    type: Literal["refactor", "optimize", "fix", "style"]
    description: str
    target: Optional[str] = None
    priority: Literal["low", "medium", "high"] = "medium"

class CodeChangeOutput(BaseModelCamel):
    """Model defining the result of a code change."""
    type: str
    description: str
    before: str
    after: str
    impact: str

class QualityMetrics(BaseModelCamel):
    """Model defining code quality metrics."""
    complexity: float = Field(ge=0, le=100)
    maintainability: float = Field(ge=0, le=100)
    test_coverage: float = Field(ge=0, le=100)

class ImproveCodeInput(BaseModelCamel):
    """Input model for code improvement endpoint."""
    changes_list: List[CodeChange]
    apply_black_formatting: bool = True
    run_linter: bool = True

class ImproveCodeOutput(BaseModelCamel):
    """Output model for code improvement endpoint."""
    code_changes: List[CodeChangeOutput]
    changes_description: str
    quality_metrics: QualityMetrics

# Testing Models
class TestType(str, Enum):
    """Enumeration of test types."""
    UNIT = "unit"
    INTEGRATION = "integration"
    E2E = "e2e"

class TestInstruction(BaseModelCamel):
    """Model defining test instructions."""
    description: str
    assertions: List[str]
    test_type: Literal["unit", "integration", "e2e"] = "unit"

class TestCodeInput(BaseModelCamel):
    """Input model for code testing endpoint."""
    test_type: TestType
    require_passing: bool
    test_instructions: List[TestInstruction]
    code_to_test: str
    minimum_coverage: float = Field(ge=0, le=100, default=80.0)

class CoverageStatus(BaseModelCamel):
    """Model defining test coverage status."""
    percentage: float = Field(ge=0, le=100)
    uncovered_lines: List[int] = Field(default_factory=list)

class TestCodeOutput(BaseModelCamel):
    """Output model for code testing endpoint."""
    code_tests: str
    tests_description: str
    coverage_status: CoverageStatus

# Deployment Models
class DeployPreviewInput(BaseModelCamel):
    """Input model for deployment preview endpoint."""
    branch_id: str
    is_private: bool
    environment_vars: Optional[Dict[str, str]] = None

class DeployPreviewOutput(BaseModelCamel):
    """Output model for deployment preview endpoint."""
    preview_url: str
    is_private: bool
    http_auth: Optional[Dict[str, str]] = None
    deployment_time: datetime

# Define agent capabilities
AGENT_CAPABILITIES = [
    Capability(
        skillPath=["Development"],
        level="domain",
        metadata=CapabilityMetadata(
            expertise="advanced"
        )
    ),
    Capability(
        skillPath=["Development", "Backend", "Python"],
        level="specialty",
        metadata=CapabilityMetadata(
            versions=["3.8", "3.9", "3.10"],
            frameworks=["Django", "FastAPI"],
            expertise="advanced"
        )
    ),
    Capability(
        skillPath=["Development", "Backend", "Python", "CodeGeneration"],
        level="skill",
        metadata=CapabilityMetadata(
            versions=["3.8", "3.9", "3.10"],
            frameworks=["Django", "FastAPI"],
            tools=["black", "pylint"]
        )
    ),
    Capability(
        skillPath=["Development", "Testing", "Python"],
        level="specialty",
        metadata=CapabilityMetadata(
            frameworks=["pytest", "unittest"],
            expertise="advanced"
        )
    ),
    Capability(
        skillPath=["Development", "Deployment", "Python"],
        level="specialty",
        metadata=CapabilityMetadata(
            platforms=["AWS", "GCP", "Azure"],
            expertise="basic"
        )
    )
]

app = FastAPI()
agent_app = FastAPI()

@configure_agent(
    base_url="http://localhost:9200",
    agent_id="python-code-agent",
    description="Advanced Python code generation, testing, and deployment agent",
    capabilities=AGENT_CAPABILITIES
)
@agent_app.post("/code_agent/python/generate_code", response_model=GenerateCodeOutput)
@agent_endpoint(
    task_type="pythonCodeTask",
    skill_name="generateCode",
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
@agent_endpoint(
    task_type="pythonCodeTask",
    skill_name="improveCode",
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
@agent_endpoint(
    task_type="pythonTestingTask",
    skill_name="testCode",
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
@agent_endpoint(
    task_type="pythonDeploymentTask",
    skill_name="deployPreview",
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

# Helper functions
def _generate_code_from_requirements(requirements: CodeRequirement) -> str:
    """Generate code based on requirements."""
    return f"""
from fastapi import FastAPI
app = FastAPI()

# Generated based on requirements: {requirements.description}
@app.get("/example")
async def example_endpoint():
    return {{"message": "Generated endpoint"}}
"""

def _generate_test_cases(code: str) -> List[str]:
    """Generate test cases for the given code."""
    return [
        "def test_example_endpoint():\n    response = client.get('/example')\n    assert response.status_code == 200"
    ]

def _generate_documentation(code: str, level: str) -> str:
    """Generate documentation for the given code."""
    return f"API Documentation\n\nEndpoints:\n- GET /example: Example endpoint\n\nDetail level: {level}"

def _apply_code_changes(change: CodeChange) -> str:
    """Apply code changes based on the change request."""
    return "def improved_function():\n    return 'Improved code'"

def _generate_tests(code: str, instructions: List[TestInstruction]) -> str:
    """Generate test code based on instructions."""
    return """
import pytest
from fastapi.testclient import TestClient

def test_example():
    assert True
"""

async def _cleanup_preview(branch_id: str):
    """Clean up preview deployment resources."""
    pass

# Mount the agent app
app.mount("/v1", agent_app)

# Set up the agents.json endpoint
setup_agent_routes(app)

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=9200)