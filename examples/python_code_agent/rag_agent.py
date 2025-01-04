from fastapi import FastAPI, HTTPException, BackgroundTasks
from pydantic import BaseModel, Field
from typing import List, Optional, Dict, Any, Literal
from datetime import datetime
import asyncio
import asyncpg
from openai import AsyncOpenAI
import pydantic_core
from contextlib import asynccontextmanager

from manifest_generator import (
    configure_agent, agent_action, setup_agent_routes,
    ActionType, Capability
)

# Models
class SearchQuery(BaseModel):
    """Input model for RAG search queries."""
    query: str = Field(..., description="The search query to process")
    top_k: int = Field(default=5, description="Number of results to return")
    context_window: Optional[int] = Field(default=None, description="Context window size")

class SearchResult(BaseModel):
    """Model for individual search results."""
    url: str
    title: str
    content: str
    relevance_score: float

class SearchResponse(BaseModel):
    """Output model for RAG search responses."""
    results: List[SearchResult]
    query_embedding_time: float
    search_time: float
    total_time: float

class RAGChatInput(BaseModel):
    """Input model for RAG-enhanced chat."""
    message: str
    context: Optional[str] = None
    history: Optional[List[Dict[str, str]]] = None

class RAGChatOutput(BaseModel):
    """Output model for RAG-enhanced chat responses."""
    response: str
    sources: List[SearchResult]
    confidence: float
    suggested_followup: Optional[List[str]] = None

# Database connection
@asynccontextmanager
async def get_db_pool():
    """Create and manage database connection pool."""
    pool = await asyncpg.create_pool(
        'postgresql://postgres:postgres@localhost:54320/pydantic_ai_rag'
    )
    try:
        yield pool
    finally:
        await pool.close()

# Initialize FastAPI app for RAG agent
rag_app = FastAPI()
RAG_CAPABILITIES = [
    Capability(
        skill_path=["Search", "RAG", "VectorSearch"],
        metadata={
            "expertise": "advanced",
            "features": ["Embedding Generation", "Similarity Search", "Relevance Scoring"],
            "databases": ["pgvector"],
            "performance": ["High-Dimensional Search", "HNSW Indexing"]
        }
    ),
    Capability(
        skill_path=["Search", "RAG", "Chat"],
        metadata={
            "expertise": "advanced",
            "models": ["Qwen-2.5"],
            "features": ["Context-Aware Responses", "Source Attribution", "Follow-up Generation"],
            "interaction_types": ["Q&A", "Documentation Search", "Technical Support"]
        }
    )
]

# Configure RAG agent
rag_app = configure_agent(
    app=rag_app,
    base_url="http://localhost:9200",
    name="RAG Assistant",
    version="1.0.0",
    description="Advanced RAG-based search and chat agent",
    capabilities=RAG_CAPABILITIES,
)


@rag_app.post("/rag_agent/search", response_model=SearchResponse)
@agent_action(
    action_type=ActionType.GENERATE,
    name="Vector Search",
    description="Perform vector search on documents",
    schema_definitions={
        "SearchQuery": SearchQuery,
        "SearchResult": SearchResult,
        "SearchResponse": SearchResponse
    },
    examples={
        "validRequests": [
            {
                "query": "How to configure logging?",
                "top_k": 5
            }
        ]
    }
)
async def vector_search(query: SearchQuery) -> SearchResponse:
    """Perform vector search on document embeddings."""
    try:
        async with get_db_pool() as pool:
            openai_client = AsyncOpenAI()
            
            # Create embedding for query
            start_time = datetime.now()
            embedding_response = await openai_client.embeddings.create(
                input=query.query,
                model='text-embedding-3-small'
            )
            embedding = embedding_response.data[0].embedding
            embedding_json = pydantic_core.to_json(embedding).decode()
            embedding_time = (datetime.now() - start_time).total_seconds()

            # Perform vector search
            search_start = datetime.now()
            rows = await pool.fetch(
                '''
                SELECT url, title, content, 
                       embedding <-> $1::vector AS distance
                FROM doc_sections 
                ORDER BY embedding <-> $1::vector 
                LIMIT $2
                ''',
                embedding_json,
                query.top_k
            )
            search_time = (datetime.now() - search_start).total_seconds()

            results = [
                SearchResult(
                    url=row['url'],
                    title=row['title'],
                    content=row['content'],
                    relevance_score=1.0 - float(row['distance'])
                )
                for row in rows
            ]

            total_time = (datetime.now() - start_time).total_seconds()

            return SearchResponse(
                results=results,
                query_embedding_time=embedding_time,
                search_time=search_time,
                total_time=total_time
            )

    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

@rag_app.post("/rag_agent/chat", response_model=RAGChatOutput)
@agent_action(
    action_type=ActionType.TALK,
    name="RAG Chat",
    description="Chat with context-aware RAG assistance",
    schema_definitions={
        "RAGChatInput": RAGChatInput,
        "RAGChatOutput": RAGChatOutput,
        "SearchResult": SearchResult
    },
    examples={
        "validRequests": [
            {
                "message": "How do I implement logging in my application?",
                "context": "Python FastAPI application",
                "history": []
            }
        ]
    }
)
async def rag_chat(input_data: RAGChatInput) -> RAGChatOutput:
    """Handle RAG-enhanced chat interactions."""
    try:
        # First, get relevant context through vector search
        search_results = await vector_search(
            SearchQuery(query=input_data.message, top_k=3)
        )

        openai_client = AsyncOpenAI()
        
        # Prepare context for chat
        context = "\n\n".join(
            f"From {result.title}:\n{result.content}"
            for result in search_results.results
        )

        # Generate chat response with context
        chat_completion = await openai_client.chat.completions.create(
            model="gpt-4",
            messages=[
                {"role": "system", "content": f"You are a helpful assistant. Use this context to answer the question:\n\n{context}"},
                {"role": "user", "content": input_data.message}
            ]
        )

        return RAGChatOutput(
            response=chat_completion.choices[0].message.content,
            sources=search_results.results[:3],
            confidence=0.95,
            suggested_followup=[
                "Tell me more about logging configuration",
                "How can I customize the log format?",
                "What are best practices for logging?"
            ]
        )

    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))