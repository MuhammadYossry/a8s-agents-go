from typing import Optional, List, Dict, Literal
from pydantic import BaseModel, Field, ConfigDict
from datetime import datetime
from enum import Enum

from manifest_generator import Capability

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
    coding_style: Optional['CodingStyle'] = None

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

class ChatMessage(BaseModel):
    """Model for chat messages."""
    content: str = Field(..., description="The content of the message")
    role: str = Field(default="user", description="Role of the message sender (user/agent)")
    timestamp: datetime = Field(default_factory=datetime.now)

class ChatInput(BaseModelCamel):
    """Input model for chat endpoint."""
    message: str = Field(..., description="User's message to the agent")
    context: Optional[str] = Field(None, description="Additional context for the conversation")
    history: Optional[List[ChatMessage]] = Field(default_factory=list, description="Previous messages in the conversation")

class ChatOutput(BaseModelCamel):
    """Output model for chat endpoint."""
    response: str = Field(..., description="Agent's response to the user")
    confidence: float = Field(..., ge=0, le=1, description="Confidence score of the response")
    suggested_actions: Optional[List[str]] = Field(default_factory=list, description="Suggested next actions")
    timestamp: datetime = Field(default_factory=datetime.now)

class CollectRequirementsOutput(BaseModel):
    """Output model for requirements collection endpoint."""
    questionnaire_form: dict = Field(..., description="JSON form structure for the questionnaire")
    timestamp: datetime = Field(default_factory=datetime.now)

class RequirementsPhase(BaseModel):
    user_query: str
    answers: Optional[dict]

class CollectRequirementsInput(BaseModel):
    message: str
    history: Optional[List[RequirementsPhase]] = None

# class CollectRequirementsInput(BaseModel):
#     """Input model for requirements collection endpoint."""
#     message: str = Field(..., description="User's project description")
#     context: Optional[str] = Field(None, description="Additional context for requirements gathering")


AGENT_CAPABILITIES = [
    Capability(
        skill_path=["Development", "Python"],
        metadata={
            "proficiency": "advanced",
            "frameworks": ["Django", "FastAPI"],
            "tools": ["black", "pylint", "pytest"],
            "specializations": ["API Development", "Testing", "Code Generation"]
        }
    ),
    Capability(
        skill_path=["AI", "Code Assistant"],
        metadata={
            "proficiency": "advanced",
            "features": ["Code Analysis", "Optimization", "Test Generation"],
            "languages": ["Python"],
            "platforms": ["AWS", "GCP", "Azure"]
        }
    )
]