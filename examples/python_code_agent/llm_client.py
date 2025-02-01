# app/services/llm.py
from typing import Optional, Dict, List, Any, Union
import httpx
from pydantic import BaseModel, Field
from loguru import logger
from functools import lru_cache, wraps
import json
from datetime import datetime
import re

from functools import lru_cache
from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    model_config = SettingsConfigDict(env_file=".env", env_ignore_empty=True, extra="ignore")
    DATE_FORMAT: str = "%Y-%m-%d"
    PROJECT_NAME: str = "Python code assistant"

    LLM_PROVIDER: str = "qwen"  # or "openai", etc
    LLM_BASE_URL: str = "http://localhost:8001"
    LLM_API_KEY: str = "default-key"
    LLM_MODEL: str = "qwen-7b-chat"
    LLM_TIMEOUT: int = 300
    LLM_DRY_RUN: bool = False
    
    # Task Extraction Settings
    TASK_EXTRACTION_TEMPERATURE: float = 0.2
    PAYLOAD_GENERATION_TEMPERATURE: float = 0.1


@lru_cache
def get_settings():
    return Settings()  # type: ignore


settings = get_settings()


class LLMMessage(BaseModel):
    role: str
    content: str

class LLMRequest(BaseModel):
    model: str
    messages: List[LLMMessage]
    temperature: float = Field(default=0.7, ge=0, le=1)
    top_p: float = Field(default=0.9, ge=0, le=1)
    stream: bool = Field(default=False)
    result_format: str = Field(default="message")
    max_tokens: Optional[int] = None
    stop: Optional[List[str]] = None
    frequency_penalty: float = Field(default=0.0)
    presence_penalty: float = Field(default=0.0)

class LLMResponse(BaseModel):
    content: str
    usage: Dict[str, int] = Field(
        default_factory=lambda: {
            "prompt_tokens": 0,
            "completion_tokens": 0, 
            "total_tokens": 0
        }
    )
    model: str

def handle_llm_errors(func):
    @wraps(func)
    async def wrapper(*args, **kwargs):
        try:
            return await func(*args, **kwargs)
        except httpx.HTTPStatusError as e:
            logger.error(f"HTTP error occurred: {e.response.text}")
            if e.response.status_code == 502:
                raise RuntimeError("Backend server error (status 502)")
            elif e.response.status_code == 503:
                raise RuntimeError("Service unavailable (status 503)")
            raise RuntimeError(f"LLM API error: {e.response.status_code}")
        except httpx.RequestError as e:
            logger.error(f"Request error occurred: {str(e)}")
            raise RuntimeError(f"Network error: {str(e)}")
        except json.JSONDecodeError as e:
            logger.error(f"JSON decode error: {str(e)}")
            raise RuntimeError("Invalid JSON response from LLM")
        except Exception as e:
            logger.error(f"Unexpected error: {str(e)}")
            raise
    return wrapper

class LLMClient:
    MAX_RETRIES = 2
    RETRY_STATUSES = {502, 503, 504}

    def __init__(
        self,
        base_url: Optional[str] = None,
        api_key: Optional[str] = None,
        model: Optional[str] = None,
        dry_run: Optional[bool] = None,
    ):
        self.base_url = base_url or settings.LLM_BASE_URL
        self.api_key = api_key or settings.LLM_API_KEY
        self.model = model or settings.LLM_MODEL
        self.dry_run = dry_run if dry_run is not None else settings.LLM_DRY_RUN
        
        if self.base_url and self.base_url[-1] == '/':
            self.base_url = self.base_url[:-1]
            
        self.http_client = httpx.AsyncClient(
            timeout=getattr(settings, 'LLM_TIMEOUT', 300.0),
            headers={
                "Authorization": f"Bearer {self.api_key}",
                "Content-Type": "application/json"
            }
        )
        
        if not self.dry_run and (not self.base_url or not self.api_key or not self.model):
            raise ValueError("Missing required LLM configuration")

    def _get_dry_run_response(self, task_type: str = "default") -> Dict[str, Any]:
        dry_run_responses = {
            "task_extraction": {
                "title": "[DRY RUN] Sample Task Extraction",
                "description": "This is a simulated task extraction response",
                "requirements": {
                    "skill_path": ["Development", "Python", "FastAPI"],
                    "action_description": "Create a sample API endpoint",
                    "parameters": {"method": "POST", "path": "/sample"}
                }
            },
            "payload_generation": {
                "endpoint": "/api/v1/sample",
                "method": "POST",
                "payload": {
                    "key": "sample_value",
                    "number": 42,
                    "timestamp": datetime.now().isoformat()
                }
            },
            "default": {
                "message": "[DRY RUN] This is a simulated response",
                "timestamp": datetime.now().isoformat()
            }
        }
        return dry_run_responses.get(task_type, dry_run_responses["default"])

    # def _extract_json_from_markdown(self, content: str) -> str:
    #     """Extract JSON content from markdown code blocks."""
    #     # Remove markdown code block syntax
    #     json_match = re.search(r'```(?:json)?\s*(.*?)\s*```', content, re.DOTALL)
    #     if json_match:
    #         return json_match.group(1).strip()
    #     return content.strip()

    def _parse_llm_response(self, response_data: Dict[str, Any]) -> LLMResponse:
        logger.debug(f"Raw LLM response: {json.dumps(response_data, indent=2)}")
        
        try:
            if "choices" not in response_data or not response_data["choices"]:
                raise KeyError("No choices in response")

            choice = response_data["choices"][0]
            
            # Extract content based on response structure
            if isinstance(choice, dict):
                if "message" in choice and "content" in choice["message"]:
                    content = choice["message"]["content"]
                elif "text" in choice:
                    content = choice["text"]
                else:
                    content = choice.get("content", "")
            else:
                content = str(choice)

            # Clean up markdown if present
            # content = self._extract_json_from_markdown(content)
            
            # Parse usage information
            # usage = response_data.get("usage", {})
            # if not usage:
            #     usage = {
            #         "prompt_tokens": response_data.get("prompt_tokens", 0),
            #         "completion_tokens": response_data.get("completion_tokens", 0),
            #         "total_tokens": response_data.get("total_tokens", 0)
            #     }
            
            # # Remove cache-related fields if present
            # cleaned_usage = {
            #     "prompt_tokens": usage.get("prompt_tokens", 0),
            #     "completion_tokens": usage.get("completion_tokens", 0),
            #     "total_tokens": usage.get("total_tokens", 0)
            # }

            return LLMResponse(
                content=content,
                model=response_data.get("model", self.model)
            )
            
        except Exception as e:
            logger.error(f"Error parsing LLM response: {str(e)}")
            logger.error(f"Response data: {response_data}")
            raise RuntimeError(f"Failed to parse LLM response: {str(e)}")

    async def _make_request(self, url: str, payload: Dict[str, Any]) -> Dict[str, Any]:
        last_error = None
        for attempt in range(self.MAX_RETRIES):
            try:
                response = await self.http_client.post(url, json=payload)
                logger.debug(f"Response status: {response.status_code}")
                logger.debug(f"Response headers: {dict(response.headers)}")
                
                if response.status_code in self.RETRY_STATUSES:
                    last_error = f"Server error (status {response.status_code})"
                    logger.warning(f"Attempt {attempt + 1}/{self.MAX_RETRIES} failed: {last_error}")
                    continue
                
                response.raise_for_status()
                return response.json()
                
            except (httpx.HTTPError, json.JSONDecodeError) as e:
                last_error = str(e)
                if attempt < self.MAX_RETRIES - 1:
                    logger.warning(f"Attempt {attempt + 1}/{self.MAX_RETRIES} failed: {last_error}")
                    continue
                raise
        
        raise RuntimeError(f"All retry attempts failed. Last error: {last_error}")

    @handle_llm_errors
    async def complete(
        self,
        prompt: str,
        system_message: Optional[str] = None,
        temperature: float = 0.7,
        task_type: str = "default",
        **kwargs
    ) -> LLMResponse:
        messages = []
        if system_message:
            messages.append(LLMMessage(role="system", content=system_message))
        messages.append(LLMMessage(role="user", content=prompt))

        request = LLMRequest(
            model=self.model,
            messages=messages,
            temperature=temperature,
            stream=False,
            result_format="message",
            **{k: v for k, v in kwargs.items() if k in LLMRequest.__fields__}
        )

        if self.dry_run:
            logger.info("🤖 DRY RUN - LLM Request:")
            logger.info(f"🔷 Model: {self.model}")
            logger.info(f"🔷 System: {system_message}")
            logger.info(f"🔷 Prompt: {prompt}")
            return LLMResponse(
                content=json.dumps(self._get_dry_run_response(task_type)),
                model=self.model
            )

        request_payload = request.model_dump(exclude_none=True)
        logger.debug(f"Request payload: {json.dumps(request_payload, indent=2)}")

        response_data = await self._make_request(
            f"{self.base_url}/chat/completions",
            request_payload
        )
        return self._parse_llm_response(response_data)

    async def close(self):
        await self.http_client.aclose()

@lru_cache
def create_llm_client(
    base_url: Optional[str] = None,
    api_key: Optional[str] = None,
    model: Optional[str] = None,
    dry_run: Optional[bool] = None
) -> LLMClient:
    return LLMClient(
        base_url=base_url,
        api_key=api_key,
        model=model,
        dry_run=dry_run
    )