from fastapi import FastAPI, HTTPException
from pydantic import BaseModel, Field, ConfigDict
from typing import Optional, Dict, Any, List, Literal
from datetime import datetime
from loguru import logger
import json
import uuid
from manifest_generator import (
    configure_agent, agent_action, ActionType,
    Workflow, WorkflowStep, WorkflowStepType,
    WorkflowTransition, WorkflowDataMapping,
    Capability, ActionMetadata
)
from llm_client import create_llm_client

llm_client = create_llm_client()
V2_CAPABILITIES = [
    Capability(
        skill_path=["Development", "Code Generation"],
        metadata={
            "version": "2.0",
            "features": ["dynamic_forms", "context_aware", "session_based"],
            "languages": ["Python"],
            "frameworks": ["FastAPI", "Django", "Flask"]
        }
    )
]


CODE_GENERATION_WORKFLOW = Workflow(
    id="code_generation",
    name="Code Generation Flow", 
    description="Multi-step code generation process with form-based requirements gathering",
    steps=[
        WorkflowStep(
            id="initiate",
            type=WorkflowStepType.START,
            action="initiate_code_generation",
            transitions=[
                WorkflowTransition(
                    target="execute",
                    data_mapping=[
                        WorkflowDataMapping(
                            source_field="questionnaire_form",
                            target_field="form_data",
                            transform="user_input"  # Indicates user needs to fill the form
                        ),
                        WorkflowDataMapping(
                            source_field="session_id",
                            target_field="session_id"
                        )
                    ]
                )
            ]
        ),
        WorkflowStep(
            id="execute",
            type=WorkflowStepType.ACTION,
            action="execute_code_generation",
            transitions=[
                WorkflowTransition(target="end")
            ]
        ),
        WorkflowStep(
            id="end",
            type=WorkflowStepType.END
        )
    ],
    initial_step="initiate"
)


class InitiateRequest(BaseModel):
    message: str = Field(..., description="User's code generation request")
    context: Optional[Dict[str, Any]] = Field(default_factory=dict)

class InitiateResponse(BaseModel):
    session_id: str
    questionnaire_form: Dict[str, Any]  # Will contain the questionnaire form
    message: Optional[str] = None

class ExecuteRequest(BaseModel):
    session_id: str
    form_data: Dict[str, Any]

class ExecuteResponse(BaseModel):
    generated_code: str
    documentation: str
    test_cases: Optional[List[str]] = None
    metadata: Optional[Dict[str, Any]] = None

# Form Generation Logic
def parse_form_response(response: str) -> dict:
    """Enhanced parser for LLM form responses."""
    try:
        content = response.content

        # Extract JSON from potential markdown blocks
        if "```json" in content:
            json_str = content.split("```json")[1].split("```")[0].strip()
        elif "```" in content:
            json_str = content.split("```")[1].strip()
        else:
            json_str = content.strip()

        form_structure = json.loads(json_str)

        # Validate structure
        if not isinstance(form_structure, dict):
            raise ValueError("Response is not a JSON object")
        if "questionnaire_form" not in form_structure:
            raise ValueError("Response missing questionnaire_form key")
        if "steps" not in form_structure["questionnaire_form"]:
            raise ValueError("Response missing steps in questionnaire_form")

        return form_structure["questionnaire_form"]

    except json.JSONDecodeError as e:
        logger.error(f"JSON decode error: {str(e)}")
        logger.error(f"Raw response: {content}")
        raise HTTPException(
            status_code=400,
            detail="Failed to generate valid form structure. Please try again."
        )
    except Exception as e:
        logger.error(f"Error processing form: {str(e)}")
        logger.error(f"Raw response: {content}")
        raise HTTPException(
            status_code=400,
            detail="Failed to process the requirements form. Please try again."
        )

async def generate_code_form(query: str) -> dict:
    """Generate dynamic form based on code generation query."""
    # Define the form template
    json_template = """
    {
      "questionnaire_form": {
        "steps": [
          {
            "title": "Section Title",
            "fields": [
              {
                "type": "text|textarea|select|checkbox|radio|number",
                "name": "field_name",
                "label": "Question text",
                "placeholder": "Helper text for user",
                "validation": { "required": true|false },
                "options": [{ "label": "Option text", "value": "option_value" }]
              }
            ]
          }
        ]
      }
    }
    """

    # Enhanced prompt for code-specific requirements
    prompt = f"""You are a code generation expert. Analyze this code request and generate a detailed tailored requirements questionnaire:

Request: {query}

Generate a JSON form *inspired* but as custom to the user query these key sections:
1. Code Specifications
   - Language/framework preferences
   - Architecture requirements
   - Key functionalities needed
2. Technical Requirements
   - Performance requirements
   - Security needs
   - Integration points
3. Development Preferences
   - Code style preferences
   - Documentation level
   - Testing requirements
4. Implementation Details
   - Specific features
   - Data structures
   - API endpoints (if applicable)

Required JSON format:
{json_template}

Field type guidelines:
- Use textarea: For detailed code requirements, architecture descriptions
- Use select: For language, framework, or architecture choices
- Use checkbox: For multiple feature selections, library choices
- Use radio: For single-choice technical decisions
- Use number: For performance metrics, timeout values

Example Response for "Create a REST API for user management":
{{
  "questionnaire_form": {{
    "steps": [
      {{
        "title": "Code Specifications",
        "fields": [
          {{
            "type": "select",
            "name": "framework",
            "label": "Primary Framework",
            "validation": {{ "required": true }},
            "options": [
              {{ "label": "FastAPI", "value": "fastapi" }},
              {{ "label": "Django", "value": "django" }},
              {{ "label": "Flask", "value": "flask" }}
            ]
          }},
          {{
            "type": "textarea",
            "name": "architecture_requirements",
            "label": "Architecture Requirements",
            "placeholder": "Describe your architectural needs (e.g., microservices, monolith)",
            "validation": {{ "required": true }}
          }}
        ]
      }},
      {{
        "title": "Technical Requirements",
        "fields": [
          {{
            "type": "checkbox",
            "name": "security_features",
            "label": "Security Requirements",
            "options": [
              {{ "label": "JWT Authentication", "value": "jwt" }},
              {{ "label": "Role-based Access", "value": "rbac" }},
              {{ "label": "Rate Limiting", "value": "rate_limit" }}
            ]
          }}
        ]
      }}
    ]
  }}
}}

Generate questions that are:
1. tailored and custom to the code generation request, be creative
2. Technical and implementation-focused
3. Include relevant technical options
4. Progress from basic to advanced requirements
5. Include helpful technical examples in placeholders

Return only the valid JSON form structure without additional text.
"""

    response = await llm_client.complete(
        prompt=prompt,
        system_message="""You are a code generation expert who creates structured requirement forms.
Follow the chain of thought process to understand the technical needs.
Output only a strict JSON form structure.""",
        temperature=0.3
    )
    
    return parse_form_response(response)

# Session Management
class SessionManager:
    def __init__(self):
        self.sessions: Dict[str, Dict[str, Any]] = {}

    def create_session(self) -> str:
        session_id = str(uuid.uuid4())
        self.sessions[session_id] = {
            "created_at": datetime.now(),
            "context": {}
        }
        return session_id

    def get_session(self, session_id: str) -> Optional[Dict[str, Any]]:
        return self.sessions.get(session_id)

    def update_session(self, session_id: str, context: Dict[str, Any]):
        if session_id in self.sessions:
            self.sessions[session_id]["context"].update(context)

    def close_session(self, session_id: str):
        self.sessions.pop(session_id, None)

# Initialize
v2_app = configure_agent(
    app=FastAPI(),
    base_url="http://localhost:9200",
    name="Python Code Assistant V2",
    version="2.0.0",
    description="Advanced code generation with dynamic forms",
    capabilities=V2_CAPABILITIES,
    workflows=[CODE_GENERATION_WORKFLOW]
)
session_manager = SessionManager()

@v2_app.post("/workflow/{workflow_id}/start")
async def start_workflow(workflow_id: str, initial_data: Dict[str, Any]):
    """Start a new workflow instance."""
    session_id = session_manager.create_session()
    # Store workflow state
    session_manager.update_session(session_id, {
        "workflow_id": workflow_id,
        "current_step": CODE_GENERATION_WORKFLOW.initial_step,
        "data": initial_data
    })
    print("ddddd")
    # Execute initial step (initiate)
    if initial_data.get("message"):
        return await initiate_code_generation(
            InitiateRequest(
                message=initial_data["message"],
                context=initial_data.get("context", {})
            )
        )
    raise HTTPException(status_code=400, detail="Missing required data")

@v2_app.post("/workflow/{workflow_id}/step/{step_id}")
async def execute_workflow_step(
    workflow_id: str,
    step_id: str,
    session_id: str,
    step_data: Dict[str, Any]
):
    """Execute a specific workflow step."""
    session = session_manager.get_session(session_id)
    if not session:
        raise HTTPException(status_code=404, detail="Session not found") 
    if step_id == "execute":
        return await execute_code_generation(
            ExecuteRequest(
                session_id=session_id,
                form_data=step_data
            )
        )
    raise HTTPException(status_code=400, detail="Invalid step")

@v2_app.post("/agents/python-code-assistant/actions/generate-code/initiate")
@agent_action(
    action_type=ActionType.QUESTION,
    name="Initiate Code Generation",
    description="Start code generation process and get dynamic form",
    response_template_md=None,
    workflow_id="code_generation",
    step_id="initiate"
)
async def initiate_code_generation(request: InitiateRequest) -> InitiateResponse:
    """Initiate code generation with dynamic form generation."""
    try:
        session_id = session_manager.create_session()
        
        # Store initial context
        session_manager.update_session(session_id, {
            "query": request.message,
            "initial_context": request.context
        })
        
        # Generate dynamic form based on query
        form = await generate_code_form(request.message)
        
        return InitiateResponse(
            session_id=session_id,
            questionnaire_form=form,
            message="Please provide the required technical details"
        )
    except Exception as e:
        logger.error(f"Error in initiate: {str(e)}")
        raise HTTPException(status_code=400, detail=str(e))

@v2_app.post("/agents/python-code-assistant/actions/generate-code/execute")
@agent_action(
    action_type=ActionType.GENERATE,
    name="Execute Code Generation",
    description="Generate code based on form input",
    response_template_md="templates/code_generation.md",
    workflow_id="code_generation",
    step_id="execute"
)
async def execute_code_generation(request: ExecuteRequest) -> ExecuteResponse:
    """Execute code generation using form data."""
    try:
        session = session_manager.get_session(request.session_id)
        if not session:
            raise HTTPException(status_code=404, detail="Invalid session")

        # Combine context with form data
        full_context = {
            **session["context"],
            "form_data": request.form_data
        }

        # Generate code (implementation details to be added)
        # This would use the form data to generate appropriate code
        
        session_manager.close_session(request.session_id)
        
        return ExecuteResponse(
            generated_code="# Generated code will go here",
            documentation="Documentation will go here",
            test_cases=["Test cases will go here"],
            metadata={"timestamp": datetime.now().isoformat()}
        )
    except Exception as e:
        logger.error(f"Error in execute: {str(e)}")
        raise HTTPException(status_code=400, detail=str(e))
