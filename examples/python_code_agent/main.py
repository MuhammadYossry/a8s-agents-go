# main.py
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
    Capability, CapabilityMetadata
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

# Model definitions for code generation
class CodingStyle(BaseModelCamel):
    patterns: Optional[List[str]] = []
    conventions: Optional[List[str]] = []

class StyleGuide(BaseModelCamel):
    formatting: Optional[Literal["black", "autopep8"]] = "black"
    max_line_length: Optional[int] = 88

class CodeRequirement(BaseModelCamel):
    language: Literal["Python"]
    framework: Literal["FastAPI", "Django"]
    description: str
    requirements: List[str]
    required_functions: List[str]
    testing_requirements: List[str]
    coding_style: Optional[CodingStyle]

class GenerateCodeInput(BaseModelCamel):
    code_requirements: CodeRequirement
    style_guide: Optional[StyleGuide]
    include_tests: bool = True
    documentation_level: Literal["minimal", "standard", "detailed"] = "standard"

class GenerateCodeOutput(BaseModelCamel):
    generated_code: str
    description: str
    test_cases: List[str]
    documentation: str

# Model definitions for code improvement
class CodeChange(BaseModelCamel):
    type: Literal["refactor", "optimize", "fix", "style"]
    description: str
    target: Optional[str]

class CodeChangeOutput(BaseModelCamel):
    type: str
    description: str
    before: str
    after: str

class QualityMetrics(BaseModelCamel):
    complexity: float
    maintainability: float
    test_coverage: float

class ImproveCodeInput(BaseModelCamel):
    changes_list: List[CodeChange]
    apply_black_formatting: bool = True
    run_linter: bool = True

class ImproveCodeOutput(BaseModelCamel):
    code_changes: List[CodeChangeOutput]
    changes_description: str
    quality_metrics: QualityMetrics

# Model definitions for testing
class TestType(str, Enum):
    UNIT = "unit"
    INTEGRATION = "integration"
    PERFORMANCE = "performance"

class TestResult(BaseModelCamel):
    passed: bool
    execution_time: float
    coverage_percentage: float
    failing_tests: List[str] = []

class TestInstruction(BaseModelCamel):
    description: str
    assertions: List[str]

class TestCodeInput(BaseModelCamel):
    test_type: Literal["unit", "integration", "e2e"]
    require_passing: bool
    test_instructions: List[TestInstruction]
    code_to_test: str
    minimum_coverage: float = 80.0

class CoverageStatus(BaseModelCamel):
    percentage: float
    uncovered_lines: List[int]

class TestCodeOutput(BaseModelCamel):
    code_tests: str
    tests_description: str
    coverage_status: CoverageStatus

# Model definitions for deployment
class DeployPreviewInput(BaseModelCamel):
    branch_id: str
    is_private: bool
    environment_vars: Optional[Dict[str, str]]

class DeployPreviewOutput(BaseModelCamel):
    preview_url: str
    is_private: bool
    http_auth: Optional[Dict[str, str]]
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
    base_url="http://localhost:9200/v1",
    agent_id="python-code-agent",
    description="Advanced Python code generation, testing, and deployment agent",
    capabilities=AGENT_CAPABILITIES
)
@agent_app.post("/code_agent/python/generate_code", response_model=GenerateCodeOutput)
@agent_endpoint(
    task_type="pythonCodeTask",
    skill_name="generateCode",
    description="Generates Python code based on requirements",
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
    try:
        generated_code = _generate_code_from_requirements(input_data.code_requirements)
        test_cases = _generate_test_cases(generated_code) if input_data.include_tests else None
        
        return GenerateCodeOutput(
            generated_code=generated_code,
            description="Generated code sample",
            test_cases=test_cases,
            documentation="Generated FastAPI endpoints with standard API patterns"
        )
    except Exception as e:
        raise HTTPException(status_code=400, detail=str(e))

@agent_app.post("/code_agent/python/improve_code", response_model=ImproveCodeOutput)
@agent_endpoint(
    task_type="pythonCodeTask",
    skill_name="improveCode",
    description="Improves and formats existing Python code",
    examples={
        "validRequests": [
            {
                "changesList": [{
                    "filePath": "main.py",
                    "originalCode": "def hello():\n  print('hello')",
                    "proposedChanges": "def hello():\n    print('hello')",
                    "changeType": "improvement",
                    "priority": "medium"
                }],
                "applyBlackFormatting": True,
                "runLinter": True
            }
        ]
    }
)
async def improve_code(input_data: ImproveCodeInput) -> ImproveCodeOutput:
    try:    
        improved_changes = []
        for change in input_data.changes_list:
            improved_code = change.proposed_changes
            
            if input_data.apply_black_formatting:
                improved_code = black.format_str(improved_code, mode=black.FileMode())
            
            if input_data.run_linter:
                lint_score = _run_linter(improved_code)
            else:
                lint_score = None

            improved_changes.append(CodeChange(
                file_path=change.file_path,
                original_code=change.original_code,
                proposed_changes=improved_code,
                change_type=change.change_type,
                priority=change.priority
            ))

        return ImproveCodeOutput(
            code_changes=improved_changes,
            changes_description="Code improvements applied successfully",
            quality_metrics={"lintScore": lint_score if lint_score else 100}
        )
    except Exception as e:
        raise HTTPException(status_code=400, detail=str(e))

@agent_app.post("/code_agent/python/test_code", response_model=TestCodeOutput)
@agent_endpoint(
    task_type="pythonTestingTask",
    skill_name="testCode",
    description="Generates and runs tests for Python code",
    examples={
        "validRequests": [
            {
                "testType": "unit",
                "requirePassing": True,
                "testInstructions": "Test all function endpoints",
                "codeToTest": "def add(a, b):\n    return a + b",
                "minimumCoverage": 80.0
            }
        ]
    }
)
async def test_code(input_data: TestCodeInput) -> TestCodeOutput:
    try:
        generated_tests = _generate_tests(input_data.code_to_test, input_data.test_type)
        test_result = _run_tests(generated_tests, input_data.code_to_test)
        
        if input_data.require_passing and not test_result.passed:
            raise HTTPException(status_code=400, detail="Tests failed")
        
        return TestCodeOutput(
            code_tests=generated_tests,
            tests_description="Generated and executed test suite",
            coverage_status=test_result
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

# Helper methods
def _generate_code_from_requirements(requirements: CodeRequirement) -> str:
    # Basic implementation that generates a simple API endpoint
    code = f"""
        from fastapi import FastAPI
        from pydantic import BaseModel

        app = FastAPI()

        {_generate_functions(requirements.required_functions)}
        """
    return code


def _generate_functions(required_functions: List[str]) -> str:
    # Helper method to generate function implementations
    function_implementations = []
    for func_name in required_functions:
        if func_name == "get_user":
            function_implementations.append("""
            @app.get("/user/{user_id}")
            async def get_user(user_id: int):
                return {"user_id": user_id, "message": "User retrieved"}
            """)
        elif func_name == "create_user":
            function_implementations.append("""
            class UserCreate(BaseModel):
                username: str
                email: str

            @app.post("/user/")
            async def create_user(user: UserCreate):
                return {"username": user.username, "message": "User created"}
            """)
    return "\n".join(function_implementations)

def _generate_test_cases(code: str) -> List[str]:
    # Basic test case implementation
    return [
        "def test_get_user():\n    response = client.get('/user/1')\n    assert response.status_code == 200",
        "def test_create_user():\n    response = client.post('/user/', json={'username': 'test', 'email': 'test@example.com'})\n    assert response.status_code == 200"
    ]

def _generate_documentation(code: str, level: str) -> Dict[str, str]:
    # Basic documentation implementation
    return {
        "overview": "Generated FastAPI endpoints for user management",
        "usage": "Run the server and access the endpoints via HTTP requests",
        "endpoints": "GET /user/{user_id}, POST /user/"
    }

def _run_linter(code: str) -> float:
    # Implementation for running pylint
    pass

def _generate_tests(code: str, test_type: TestType) -> List[str]:
    # Implementation for test generation
    pass

def _run_tests(tests: List[str], code: str) -> TestResult:
    # Implementation for test execution
    pass

async def _cleanup_preview(branch_id: str):
    # Implementation for cleanup task
    pass

# Using Request instance
# agent_app.get("/url-list")
# def get_all_urls_from_request(request: Request):
url_list = [
    {"path": route.path, "name": route.name} for route in app.routes
]


# Mount the agent app
app.mount("/v1", agent_app)

# Add the agents.json endpoint
setup_agent_routes(app)

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=9200)