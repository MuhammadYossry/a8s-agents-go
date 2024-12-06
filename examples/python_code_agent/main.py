from fastapi import FastAPI, HTTPException, BackgroundTasks, Request
from pydantic import BaseModel, Field, ConfigDict
from typing import List, Optional, Dict, Any, Literal
from enum import Enum
import ast
import black
import pylint.lint
from dataclasses import dataclass
import json
from datetime import datetime

from manifest_generator import setup_agent_routes, agent_endpoint

class BaseModelCamel(BaseModel):
    """Base model that configures camelCase support."""
    model_config = ConfigDict(
        alias_generator=lambda s: ''.join(
            word.capitalize() if i > 0 else word
            for i, word in enumerate(s.split('_'))
        ),
        populate_by_name=True
    )

# Base Models for Code Management
class CodeRequirement(BaseModelCamel):
    description: str
    required_functions: List[str]
    dependencies: List[str] = []
    python_version: str = "3.9"
    testing_requirements: Optional[List[str]]

class GenerateCodeInput(BaseModelCamel):
    code_requirements: CodeRequirement
    style_guide: Optional[str] = "PEP8"
    include_tests: Optional[bool] = False
    documentation_level: Literal["minimal", "standard", "detailed"] = "standard"

class GenerateCodeOutput(BaseModelCamel):
    generated_code: str
    description: str
    test_cases: Optional[List[str]]
    documentation: Dict[str, str]

class CodeChange(BaseModelCamel):
    file_path: str
    original_code: str
    proposed_changes: str
    change_type: Literal["improvement", "bug_fix", "feature", "refactor"]
    priority: Literal["low", "medium", "high"]

class ImproveCodeInput(BaseModelCamel):
    changes_list: List[CodeChange]
    apply_black_formatting: bool = True
    run_linter: bool = True

class ImproveCodeOutput(BaseModelCamel):
    code_changes: List[CodeChange]
    changes_description: str
    quality_metrics: Dict[str, float]

class TestType(str, Enum):
    UNIT = "unit"
    INTEGRATION = "integration"
    PERFORMANCE = "performance"

class TestCodeInput(BaseModelCamel):
    test_type: TestType
    require_passing: bool
    test_instructions: str
    code_to_test: str
    minimum_coverage: float = 80.0

class TestResult(BaseModelCamel):
    passed: bool
    execution_time: float
    coverage_percentage: float
    failing_tests: List[str] = []

class TestCodeOutput(BaseModelCamel):
    code_tests: List[str]
    tests_description: str
    coverage_status: TestResult

class DeployPreviewInput(BaseModelCamel):
    branch_id: str
    is_private: bool
    environment_vars: Optional[Dict[str, str]]

class DeployPreviewOutput(BaseModelCamel):
    preview_url: str
    is_private: bool
    http_auth: Optional[Dict[str, str]]
    deployment_time: datetime
app = FastAPI()
agent_app = FastAPI()

@agent_app.post("/code_agent/python/generate_code", response_model=GenerateCodeOutput)
@agent_endpoint(
    task_type="pythonCodeTask",
    skill_name="generateCode",
    description="Generates Python code based on requirements"
)
async def generate_code(input_data: GenerateCodeInput) -> GenerateCodeOutput:
    try:
        # Simulate code generation based on requirements
        generated_code = _generate_code_from_requirements(input_data.code_requirements)
        test_cases = _generate_test_cases(generated_code) if input_data.include_tests else None
        
        return GenerateCodeOutput(
            generated_code=generated_code,
            description="Generated code sample",
            test_cases=["test1", "test2"],
            documentation={"overview": "Sample documentation"}
        )
    except Exception as e:
        raise HTTPException(status_code=400, detail=str(e))

@agent_app.post("/code_agent/python/improve_code", response_model=ImproveCodeOutput)
@agent_endpoint(
    task_type="pythonCodeTask",
    skill_name="improveCode",
    description="Improves and formats existing Python code"
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
            code_changes=[],
            changes_description="Sample changes",
            quality_metrics={"score": 100}
        )
    except Exception as e:
        raise HTTPException(status_code=400, detail=str(e))

@agent_app.post("/code_agent/python/test_code", response_model=TestCodeOutput)
@agent_endpoint(
    task_type="pythonTestingTask",
    skill_name="testCode",
    description="Generates and runs tests for Python code"
)
async def test_code(input_data: TestCodeInput) -> TestCodeOutput:
    try:
        # Generate and run tests based on input
        generated_tests = _generate_tests(input_data.code_to_test, input_data.test_type)
        test_result = _run_tests(generated_tests, input_data.code_to_test)
        
        if input_data.require_passing and not test_result.passed:
            raise HTTPException(status_code=400, detail="Tests failed")
        
        return TestCodeOutput(
            code_tests=["test1"],
            tests_description="Sample tests",
            coverage_status=TestResult(
                passed=True,
                execution_time=1.0,
                coverage_percentage=100.0,
                failing_tests=[]
            )
        )
    except Exception as e:
        raise HTTPException(status_code=400, detail=str(e))

@agent_app.post("/code_agent/python/deploy_preview", response_model=DeployPreviewOutput)
@agent_endpoint(
    task_type="pythonDeploymentTask",
    skill_name="deployPreview",
    description="Creates a preview deployment for code review"
)   
async def deploy_preview(
    input_data: DeployPreviewInput, background_tasks: BackgroundTasks
) -> DeployPreviewOutput:
    try:
        # Simulate deployment process
        preview_url = f"https://preview.example.com/{input_data.branch_id}"
        http_auth = {"username": "preview", "password": "secret"} if input_data.is_private else None
        
        # Add cleanup task to background tasks
        background_tasks.add_task(_cleanup_preview, input_data.branch_id)
        
        return DeployPreviewOutput(
            preview_url="https://example.com",
            is_private=True,
            http_auth={"username": "test"},
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
print(url_list)
# Prints: [{'path': '/openapi.json', 'name': 'openapi'}, {'path': '/docs', 'name': 'swagger_ui_html'}, {'path': '/docs/oauth2-redirect', 'name': 'swagger_ui_redirect'}, {'path': '/redoc', 'name': 'redoc_html'}]


# Mount the agent app
app.mount("/v1", agent_app)

# Add the agents.json endpoint
setup_agent_routes(app)

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=9200)