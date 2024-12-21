from fastapi import FastAPI, Request
from pydantic import BaseModel
from typing import Dict, Any, List, Optional
import logging
import json
from datetime import datetime

# Configure logging with a more detailed format
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s',
    datefmt='%Y-%m-%d %H:%M:%S'
)
logger = logging.getLogger(__name__)

app = FastAPI()

# Pydantic models to match the Go structures
class PayloadModel(BaseModel):
    action: str
    parameters: Dict[str, Any]

class EventModel(BaseModel):
    id: str
    type: str
    timestamp: datetime
    payload: PayloadModel

# Dapr pub/sub subscription
@app.get("/dapr/subscribe")
async def subscribe() -> Dict[str, List[Dict[str, str]]]:
    logger.info("🔔 Dapr subscription endpoint called - Setting up subscription for request-generator topic")
    return {
        "subscriptions": [
            {
                "pubsubname": "internal-agents",
                "topic": "request-generator",
                "route": "/request-generator"
            }
        ]
    }

# Route to handle incoming events
@app.post("/request-generator")
async def handle_request_generator(request: Request) -> Dict[str, Any]:
    """Handler for request-generator messages."""
    try:
        # Get the raw request body
        cloud_event = await request.json()
        logger.info("📦 Received cloud event from Dapr")
        
        # The actual event data is nested in the cloud event's data field
        event_data = cloud_event.get('data', {})
        
        # Parse the event data using our Pydantic model
        event = EventModel(**event_data)
        
        # Print detailed event information
        logger.info("🎯 Event Details:")
        logger.info(f"  ID: {event.id}")
        logger.info(f"  Type: {event.type}")
        logger.info(f"  Timestamp: {event.timestamp}")
        logger.info("  Payload:")
        logger.info(f"    Action: {event.payload.action}")
        logger.info(f"    Parameters: {event.payload.parameters}")

        return {
            "success": True,
            "message": f"✅ Successfully processed event {event.id}"
        }

    except Exception as e:
        logger.error(f"❌ Error processing event: {str(e)}", exc_info=True)
        return {
            "success": False,
            "error": str(e)
        }

# Health check endpoint
@app.get("/healthz")
async def health_check() -> Dict[str, str]:
    return {"status": "healthy"}

if __name__ == "__main__":
    logger.info("🚀 Starting Dapr event consumer service...")
    import uvicorn
    uvicorn.run(
        app, 
        host="0.0.0.0", 
        port=9100,
        log_level="info"
    )