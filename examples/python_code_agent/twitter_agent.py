from fastapi import FastAPI, HTTPException
from pydantic import BaseModel, Field
from typing import List, Optional, Dict, Any
from datetime import datetime
from enum import Enum

from manifest_generator import configure_agent, agent_action, ActionType, Capability

# Models
class TweetInput(BaseModel):
    text: str
    reply_to_id: Optional[str] = None
    poll_options: Optional[List[str]] = None
    poll_duration_minutes: Optional[int] = Field(default=1440, ge=1, le=10080)

class TweetOutput(BaseModel):
    id: str
    text: str
    created_at: datetime
    metrics: Optional[Dict[str, int]] = None

class SearchRequest(BaseModel):
    query: str
    limit: int = Field(default=10, ge=1, le=100)
    include_metrics: bool = False

class Tweet(BaseModel):
    id: str
    text: str
    author_id: str
    created_at: datetime
    public_metrics: Optional[Dict[str, int]] = None

class SearchResponse(BaseModel):
    tweets: List[Tweet]
    next_token: Optional[str] = None

# Twitter Agent capabilities
TWITTER_CAPABILITIES = [
    Capability(
        skill_path=["Social", "Twitter", "Engagement"],
        metadata={
            "expertise": "advanced",
            "features": [
                "Tweet Creation",
                "Poll Creation",
                "Tweet Search",
                "Engagement Analytics"
            ],
            "api_versions": ["v2"],
            "auth_methods": ["OAuth 2.0"]
        }
    )
]

# Initialize FastAPI app
twitter_app = FastAPI()

# Configure Twitter agent
twitter_app = configure_agent(
    app=twitter_app,
    base_url="http://localhost:9200",
    name="Twitter Assistant",
    version="1.0.0",
    description="Twitter interaction and analytics agent",
    capabilities=TWITTER_CAPABILITIES
)

class TwitterAgent:
    def __init__(self):
        self.bearer_token = "YOUR_BEARER_TOKEN"
        
    async def create_tweet(self, tweet_input: TweetInput) -> TweetOutput:
        # Simulated tweet creation
        return TweetOutput(
            id="1234567890",
            text=tweet_input.text,
            created_at=datetime.now(),
            metrics={"impressions": 0, "likes": 0, "retweets": 0}
        )
        
    async def search_tweets(self, query: str, limit: int) -> SearchResponse:
        # Simulated tweet search
        sample_tweets = [
            Tweet(
                id=str(i),
                text=f"Sample tweet #{i} matching '{query}'",
                author_id="123456",
                created_at=datetime.now()
            )
            for i in range(limit)
        ]
        return SearchResponse(tweets=sample_tweets)

# Initialize agent
agent = TwitterAgent()

@twitter_app.post("/twitter/tweet", response_model=TweetOutput)
@agent_action(
    action_type=ActionType.GENERATE,
    name="Create Tweet",
    description="Create a new tweet with optional poll",
    schema_definitions={
        "TweetInput": TweetInput,
        "TweetOutput": TweetOutput
    },
    examples={
        "validRequests": [
            {
                "text": "Hello Twitter! This is a poll.",
                "poll_options": ["Yes", "No", "Maybe"],
                "poll_duration_minutes": 1440
            }
        ]
    }
)
async def create_tweet(tweet_input: TweetInput) -> TweetOutput:
    """Create a new tweet with optional poll."""
    try:
        return await agent.create_tweet(tweet_input)
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

@twitter_app.post("/twitter/search", response_model=SearchResponse)
@agent_action(
    action_type=ActionType.GENERATE,
    name="Search Tweets",
    description="Search for tweets using query",
    schema_definitions={
        "SearchRequest": SearchRequest,
        "SearchResponse": SearchResponse,
        "Tweet": Tweet
    },
    examples={
        "validRequests": [
            {
                "query": "#Python",
                "limit": 10,
                "include_metrics": True
            }
        ]
    }
)
async def search_tweets(search_request: SearchRequest) -> SearchResponse:
    """Search for tweets matching query."""
    try:
        return await agent.search_tweets(search_request.query, search_request.limit)
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(twitter_app, host="0.0.0.0", port=9200)