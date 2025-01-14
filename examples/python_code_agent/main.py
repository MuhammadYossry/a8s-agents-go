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

from manifest_generator import setup_agent_routes
# Import and Mount agent apps
from code_agent import agent_app as code_agent_app
from rag_agent import rag_app
from flight_agent import flight_app
from twitter_agent import twitter_app

app = FastAPI()
# Set all CORS enabled origins
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*", "http://localhost:8000"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

app.mount("/v1/code_agent", code_agent_app, name="code_agent")
app.mount("/v1/rag_agent", rag_app, name="rag_agent")
app.mount("/v1/flight_agent", flight_app, name="flight_agent")
app.mount("/v1/twitter_agent", flight_app, name="twitter_agent")

# Set up the agents.json endpoint and other routes
setup_agent_routes(app)

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=9200)