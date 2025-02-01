from fastapi import FastAPI, HTTPException, BackgroundTasks, Request
from pathlib import Path
from datetime import datetime
import json
import black
from typing import List, Dict, Any

from models import (
    ChatInput, ChatOutput, GenerateCodeInput, GenerateCodeOutput,
    ImproveCodeInput, ImproveCodeOutput, TestCodeInput, TestCodeOutput,
    DeployPreviewInput, DeployPreviewOutput, AGENT_CAPABILITIES,
    CollectRequirementsInput, CollectRequirementsOutput, RequirementsPhase
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

import logging

logger = logging.getLogger(__name__)

def parse_questionnaire_response(response: str) -> dict:
    """Clean and extract JSON from various response formats."""
    try:
        # Split response into questionnaire and JSON parts
        content = response.content

        #  Extract JSON from response
        content = response.content

        # If response is wrapped in markdown code blocks, extract just the JSON
        if "```json" in content:
            json_str = content.split("```json")[1].split("```")[0].strip()
        elif "```" in content:
            json_str = content.split("```")[1].strip()
        else:
            json_str = content.strip()

        # Parse the JSON
        form_structure = json.loads(json_str)

        # Validate expected structure
        if not isinstance(form_structure, dict):
            raise ValueError("Response is not a JSON object")
        if "questionnaire_form" not in form_structure:
            raise ValueError("Response missing questionnaire_form key")
        if "steps" not in form_structure["questionnaire_form"]:
            raise ValueError("Response missing steps in questionnaire_form")

        # Return just the form structure
        return form_structure["questionnaire_form"]

    except json.JSONDecodeError as e:
        logger.error(f"JSON decode error: {str(e)}")
        logger.error(f"Raw response: {content}")
        logger.error(f"Extracted JSON: {json_str}")
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

async def _generate_requirements_form(message: str) -> dict:
    """Generate both questionnaire and JSON form using Chain of Thought in a single prompt."""
    # Define the JSON template separately
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

    # Define the prompt without deeply nested f-strings
    prompt = f"""You are a requirements gathering expert. Analyze this project description and generate a detailed requirements questionnaire:

Project: {message}

Generate a JSON form with 5 key sections:
1. Core Purpose and Goals (focus on objectives, target users, key features)
2. Content and Features (specific functionalities, user interactions)
3. Design and User Experience (UI/UX preferences, responsive design)
4. Technical Requirements (platform, integrations, infrastructure)
5. Timeline and Budget (development timeline, budget constraints)

Required JSON format:
{json_template}

Field type guidelines:
- Use textarea: For detailed explanations (goals, requirements)
- Use select: For single-choice predefined options (platforms, design themes)
- Use checkbox: For multiple-choice selections (features, integrations)
- Use radio: For mutually exclusive choices (yes/no, priority levels)
- Use number: For budgets, timelines, quantities

Example 1:
Query: "I want to build a website for collecting user reviews on tech books, It should be similar to social book club"
Response:
{{
  "questionnaire_form": {{
    "steps": [
      {{
        "title": "Core Purpose and Goals",
        "fields": [
          {{
            "type": "textarea",
            "name": "primary_goals",
            "label": "Primary goals and objectives",
            "placeholder": "Describe the primary goals and objectives...",
            "validation": {{ "required": true }}
          }},
          {{
            "type": "textarea",
            "name": "target_audience",
            "label": "Target audience",
            "placeholder": "Describe the target audience...",
            "validation": {{ "required": true }}
          }},
          {{
            "type": "checkbox",
            "name": "key_features",
            "label": "Key features needed",
            "placeholder": "",
            "validation": {{ "required": true }},
            "options": [
              {{
                "label": "User profiles with review history",
                "value": "user_profiles"
              }},
              {{
                "label": "Book discussion forums or comment sections",
                "value": "discussion_forums"
              }},
              {{
                "label": "Book rating and review system",
                "value": "rating_system"
              }}
            ]
          }}
        ]
      }},
      {{
        "title": "Content and Features",
        "fields": [
          {{
            "type": "textarea",
            "name": "content_description",
            "label": "Describe the content and features",
            "placeholder": "Provide details about the content and features...",
            "validation": {{ "required": true }}
          }}
        ]
      }}
    ]
  }}
}}

Example shortened response:
Query: "I want to build a recipe sharing and meal planning app"
Response:
{{
  "questionnaire_form": {{
    "steps": [
      {{
        "title": "Core Purpose and Goals",
        "fields": [
          {{
            "type": "textarea",
            "name": "primary_goals",
            "label": "What are the main goals of your recipe app?",
            "placeholder": "e.g., help users discover recipes, plan meals, share cooking experiences",
            "validation": {{ "required": true }}
          }},
          {{
            "type": "checkbox",
            "name": "key_features",
            "label": "Essential features needed",
            "options": [
              {{
                "label": "Recipe creation and sharing",
                "value": "recipe_sharing"
              }},
              {{
                "label": "Meal planning calendar",
                "value": "meal_planning"
              }},
              {{
                "label": "Shopping list generation",
                "value": "shopping_list"
              }},
              {{
                "label": "Nutritional information tracking",
                "value": "nutrition_tracking"
              }}
            ]
          }}
        ]
      }},
      {{
        "title": "Content and Features",
        "fields": [
          {{
            "type": "textarea",
            "name": "content_description",
            "label": "Describe the content and features",
            "placeholder": "Provide details about the content and features...",
            "validation": {{ "required": true }}
          }}
        ]
      }}
    ]
  }}
}}

Make questions:
1. Specific to the project type (website, mobile app, platform)
2. Progressive (basic → advanced requirements)
3. Include relevant options based on industry standards
4. All critical fields should have "required": true
5. Add helpful placeholder text

Output only the valid JSON questionnaire form without any additional text.
"""

    response = await llm_client.complete(
        prompt=prompt,
        system_message="""You are a requirements analysis expert who creates structured forms.
Follow the chain of thought process carefully.
First output a markdown questionnaire, then output a strict JSON form structure.""",
        temperature=0.3
    )
    return parse_questionnaire_response(response)

async def _generate_phase2_form(initial_query: str, phase1_answers: dict) -> dict:
    prompt = f"""Given the initial project query and phase 1 answers, create a detailed technical questionnaire.

Initial Query: {initial_query}
Phase 1 Answers: {json.dumps(phase1_answers, indent=2)}

Generate *focused* and *crafted* questions based on the selected features and requirements. Include:
- Specific technical implementation details
- Detailed UX/UI requirements
- Integration specifications
- Performance requirements
- Security considerations
- Testing requirements
- Deployment preferences

Required JSON format:
{json_template}

Example 1:
shortened Response, build a new one from the first princples based on the previous questionare answers:
{{
  "questionnaire_form": {{
    "steps": [
      {{
        "title": "Core Purpose and Goals",
        "fields": [
          {{
            "type": "textarea",
            "name": "primary_goals",
            "label": "What are the main goals of your recipe app?",
            "placeholder": "e.g., help users discover recipes, plan meals, share cooking experiences",
            "validation": {{ "required": true }}
          }},
          {{
            "type": "checkbox",
            "name": "key_features",
            "label": "Essential features needed",
            "options": [
              {{
                "label": "Recipe creation and sharing",
                "value": "recipe_sharing"
              }},
              {{
                "label": "Meal planning calendar",
                "value": "meal_planning"
              }},
              {{
                "label": "Shopping list generation",
                "value": "shopping_list"
              }},
              {{
                "label": "Nutritional information tracking",
                "value": "nutrition_tracking"
              }}
            ]
          }}
        ]
      }},
      {{
        "title": "Content and Features",
        "fields": [
          {{
            "type": "textarea",
            "name": "content_description",
            "label": "Describe the content and features",
            "placeholder": "Provide details about the content and features...",
            "validation": {{ "required": true }}
          }}
        ]
      }}
    ]
  }}
}}

Output only JSON following the questionnaire_form format."""

    response = await llm_client.complete(
        prompt=prompt,
        system_message="You are a technical requirements analyst. Generate detailed follow-up questions based on initial requirements.",
        temperature=0.3
    )

    # Same response parsing as before
    return parse_questionnaire_response(response)

@agent_app.post("/code_agent/python/collect_requirements", response_model=CollectRequirementsOutput)
@agent_action(
    action_type=ActionType.QUESTION,
    name="Collect Requirements",
    description="Generates a requirements questionnaire form based on user input"
)
async def collect_requirements(input_data: CollectRequirementsInput) -> CollectRequirementsOutput:
    """Generate a structured requirements form based on the user's project description."""
    try:
        if not input_data.history:
            questionnaire_form = await _generate_requirements_form(input_data.message)
            return CollectRequirementsOutput(questionnaire_form=questionnaire_form)
        last_phase = input_data.history[-1]
        questionnaire_form = await _generate_phase2_form(input_data.message, last_phase.answers)
        return CollectRequirementsOutput(questionnaire_form=questionnaire_form)
    except Exception as e:
        logger.error(f"Error in collect_requirements: {str(e)}")
        raise HTTPException(status_code=400, detail=str(e))

# def clean_json_str(content: str) -> str:
#     """Clean and extract JSON from various response formats."""
#     # Remove any leading/trailing whitespace
#     content = content.strip()

#     # If content is wrapped in markdown code blocks, extract it
#     if "```json" in content:
#         # Split by ```json and take the part after it
#         parts = content.split("```json")
#         if len(parts) > 1:
#             content = parts[1]
#         # Split by ``` and take the part before it
#         parts = content.split("```")
#         if len(parts) > 0:
#             content = parts[0]
#     elif "```" in content:
#         # Split by ``` and take the content between first and second occurrence
#         parts = content.split("```")
#         if len(parts) > 1:
#             content = parts[1]

#     # Remove any remaining whitespace
#     return content.strip()

# async def _generate_questionnaire(message: str) -> str:
#     """Generate initial markdown questionnaire using LLM."""
#     prompt = f"""Create a focused requirements questionnaire for: {message}

# Follow this structure:
# 1. Core Purpose and Goals
#    - Primary goals and objectives
#    - Target audience
#    - Key features needed

# 2. Content and Features
#    - Main content types
#    - Key functionality
#    - User interactions

# 3. Design and User Experience
#    - Design preferences
#    - Responsive design needs
#    - Key user interactions

# 4. Technical Requirements
#    - Platform preferences
#    - Integration needs
#    - Infrastructure requirements

# 5. Timeline and Budget
#    - Project timeline
#    - Budget constraints
#    - Maintenance needs

# Use bullet points and include 2-3 examples in parentheses for each question.
# Format using markdown headers (###) for sections.
# Be concise and focus on essential information."""

#     response = await llm_client.complete(
#         prompt=prompt,
#         system_message="You are a requirements analyst. Create a focused, well-structured questionnaire that captures essential project requirements. Be concise and practical.",
#         temperature=0.3
#     )
#     return response.content

# async def _generate_json_form(questionnaire: str) -> dict:
#     """Convert markdown questionnaire to JSON form structure using LLM."""
#     prompt = f"""Convert this questionnaire into a strict JSON format:

# {questionnaire}

# IMPORTANT: Your response must be a single JSON object matching exactly this format:
# {{
#   "steps": [
#     {{
#       "title": string,
#       "fields": [
#         {{
#           "type": "text" | "textarea" | "select" | "checkbox" | "radio" | "number",
#           "name": string,
#           "label": string,
#           "placeholder": string,
#           "validation": {{ "required": boolean }},
#           "options": [{{ "label": string, "value": string }}] // only for select/checkbox/radio
#         }}
#       ]
#     }}
#   ]
# }}

# Rules:
# 1. Use "textarea" for long text inputs
# 2. Use "select" for single choice from multiple options
# 3. Use "checkbox" for multiple selections
# 4. Use "number" for numerical inputs with min/max validation
# 5. Always include "validation" object with at least "required" field

# YOUR RESPONSE MUST START WITH {{"""

#     response = await llm_client.complete(
#         prompt=prompt,
#         system_message="Generate only a valid JSON object. No explanations, no markdown, no code blocks. Response must start with { and end with }.",
#         temperature=0.2
#     )

#     try:
#         # Get clean JSON string
#         json_str = clean_json_str(response.content)

#         # Log the cleaned JSON string for debugging
#         logger.debug(f"Cleaned JSON string: {json_str}")

#         # Verify JSON starts with { and ends with }
#         if not (json_str.startswith('{') and json_str.endswith('}')):
#             raise ValueError("Response is not a valid JSON object")

#         return json.loads(json_str)

#     except json.JSONDecodeError as e:
#         logger.error(f"JSON decode error: {str(e)}")
#         logger.error(f"Problematic JSON: {json_str}")
#         raise HTTPException(
#             status_code=400,
#             detail="Failed to generate valid form structure. Please try again."
#         )
#     except Exception as e:
#         logger.error(f"Error processing form: {str(e)}")
#         raise HTTPException(
#             status_code=400,
#             detail=str(e)
#         )

# @agent_app.post("/code_agent/python/collect_requirements", response_model=CollectRequirementsOutput)
# @agent_action(
#     action_type=ActionType.QUESTION,
#     name="Collect Requirements",
#     description="Generates a requirements questionnaire and corresponding JSON form based on user input"
# )
# async def collect_requirements(input_data: CollectRequirementsInput) -> CollectRequirementsOutput:
#     """Handle requirements collection process using two-stage prompting."""
#     try:
#         # Stage 1: Generate markdown questionnaire
#         questionnaire = await _generate_questionnaire(input_data.message)

#         # Stage 2: Convert questionnaire to JSON form
#         questionnaire_form = await _generate_json_form(questionnaire)

#         return CollectRequirementsOutput(
#             questionnaire=questionnaire,
#             questionnaire_form=questionnaire_form,
#             confidence=0.85
#         )
#     except Exception as e:
#         logger.error(f"Error in collect_requirements: {str(e)}")
#         raise HTTPException(status_code=400, detail=str(e))