from fastapi import FastAPI, HTTPException, BackgroundTasks
from pydantic import BaseModel, Field
from typing import List, Optional, Dict, Any, Literal
import datetime
from enum import Enum

from manifest_generator import (
    configure_agent, agent_action, ActionType, Capability
)

class SeatClass(str, Enum):
    """Available seat classes."""
    ECONOMY = "economy"
    BUSINESS = "business"
    FIRST = "first"

# Models
class FlightDetails(BaseModel):
    """Details of a flight."""
    flight_number: str = Field(..., description="Unique flight identifier")
    price: float = Field(..., description="Flight price in USD")
    origin: str = Field(..., description="Three-letter airport code for origin")
    destination: str = Field(..., description="Three-letter airport code for destination")
    flight_date: datetime.date = Field(..., description="Flight date")

class NoFlightFound(BaseModel):
    """Response when no valid flight is found."""
    reason: str = Field(..., description="Reason why no flight was found")

class SeatPreference(BaseModel):
    """Seat preference details."""
    row: int = Field(..., ge=1, le=30, description="Row number")
    seat: Literal["A", "B", "C", "D", "E", "F"] = Field(..., description="Seat letter")
    seat_class: SeatClass = Field(default=SeatClass.ECONOMY)

class FlightSearchInput(BaseModel):
    """Input for flight search."""
    origin: str = Field(..., description="Three-letter airport code")
    destination: str = Field(..., description="Three-letter airport code")
    departure_date: datetime.date = Field(..., description="Desired flight date")
    passengers: int = Field(default=1, ge=1, le=9, description="Number of passengers")
    seat_class: Optional[SeatClass] = None

class FlightSearchOutput(BaseModel):
    """Output for flight search results."""
    flights: List[FlightDetails]
    search_time: float
    filters_applied: Dict[str, Any]

class BookingInput(BaseModel):
    """Input for flight booking."""
    flight_number: str
    passengers: List[Dict[str, str]]
    seat_preferences: List[SeatPreference]

class BookingOutput(BaseModel):
    """Output for booking confirmation."""
    booking_reference: str
    flight_details: FlightDetails
    seats: List[SeatPreference]
    total_price: float
    booking_time: datetime.datetime

# Initialize FastAPI app for flight agent
flight_app = FastAPI()

# Define rich capabilities for the flight agent
FLIGHT_CAPABILITIES = [
    Capability(
        skill_path=["Travel", "Flight", "Search"],
        metadata={
            "expertise": "advanced",
            "features": [
                "Real-time Flight Search",
                "Price Comparison",
                "Seat Selection",
                "Multi-city Routing"
            ],
            "supported_classes": ["Economy", "Business", "First"],
            "route_coverage": "Global",
            "booking_features": [
                "Instant Confirmation",
                "Seat Selection",
                "Special Requests"
            ]
        }
    ),
    Capability(
        skill_path=["Travel", "Flight", "Booking"],
        metadata={
            "expertise": "advanced",
            "features": [
                "Real-time Booking",
                "Seat Allocation",
                "Fare Rules",
                "Cancellation Policies"
            ],
            "payment_methods": ["Credit Card", "Digital Wallet"],
            "booking_window": "1-365 days",
            "passenger_types": ["Adult", "Child", "Infant"]
        }
    )
]

# Configure flight agent
flight_app = configure_agent(
    app=flight_app,
    base_url="http://localhost:9200",
    name="Flight Assistant",
    version="1.0.0",
    description="Advanced flight search and booking agent",
    capabilities=FLIGHT_CAPABILITIES
)

@flight_app.post("/flight_agent/search", response_model=FlightSearchOutput)
@agent_action(
    action_type=ActionType.GENERATE,
    name="Search Flights",
    description="Search for available flights based on criteria",
    response_template_md="templates/sample_test.md",
    schema_definitions={
        "FlightDetails": FlightDetails,
        "FlightSearchInput": FlightSearchInput
    },
    examples={
        "validRequests": [
            {
                "origin": "SFO",
                "destination": "JFK",
                "departure_date": "2025-01-15",
                "passengers": 2,
                "seat_class": "economy"
            }
        ]
    }
)
async def search_flights(input_data: FlightSearchInput) -> FlightSearchOutput:
    """Search for available flights based on search criteria."""
    try:
        # Simulate flight search from a database or external API
        # In a real implementation, this would connect to actual flight data sources
        start_time = datetime.datetime.now()
        
        # Sample flight data (would come from real data source)
        sample_flights = [
            FlightDetails(
                flight_number=f"{input_data.origin}{input_data.destination}123",
                price=299.99,
                origin=input_data.origin,
                destination=input_data.destination,
                flight_date=input_data.departure_date
            ),
            FlightDetails(
                flight_number=f"{input_data.origin}{input_data.destination}456",
                price=349.99,
                origin=input_data.origin,
                destination=input_data.destination,
                flight_date=input_data.departure_date
            )
        ]

        search_time = (datetime.datetime.now() - start_time).total_seconds()

        return FlightSearchOutput(
            flights=sample_flights,
            search_time=search_time,
            filters_applied={
                "origin": input_data.origin,
                "destination": input_data.destination,
                "date": input_data.departure_date,
                "passengers": input_data.passengers,
                "seat_class": input_data.seat_class
            }
        )
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

@flight_app.post("/flight_agent/book", response_model=BookingOutput)
@agent_action(
    action_type=ActionType.GENERATE,
    name="Book Flight",
    description="Book a flight with specified details and seat preferences",
    schema_definitions={
        "BookingInput": BookingInput,
        "BookingOutput": BookingOutput,
        "SeatPreference": SeatPreference,
        "FlightDetails": FlightDetails
    },
    examples={
        "validRequests": [
            {
                "flight_number": "SFO-JFK123",
                "passengers": [
                    {"first_name": "John", "last_name": "Doe"}
                ],
                "seat_preferences": [
                    {"row": 12, "seat": "A", "seat_class": "economy"}
                ]
            }
        ]
    }
)
async def book_flight(
    input_data: BookingInput,
    background_tasks: BackgroundTasks
) -> BookingOutput:
    """Book a flight with the specified details."""
    try:
        # Simulate flight booking process
        # In real implementation, this would interact with airline booking systems
        booking_reference = f"BK{datetime.datetime.now().strftime('%Y%m%d%H%M%S')}"
        
        # Simulate flight details retrieval
        flight_details = FlightDetails(
            flight_number=input_data.flight_number,
            price=299.99,  # Would be actual price from database
            origin="SFO",  # Would be retrieved based on flight number
            destination="JFK",
            flight_date=datetime.date(2025, 1, 15)
        )

        # Calculate total price (would include actual pricing logic)
        total_price = flight_details.price * len(input_data.passengers)

        # Add background task for confirmation email (simulated)
        background_tasks.add_task(
            lambda: print(f"Sending booking confirmation for {booking_reference}")
        )

        return BookingOutput(
            booking_reference=booking_reference,
            flight_details=flight_details,
            seats=input_data.seat_preferences,
            total_price=total_price,
            booking_time=datetime.datetime.now()
        )
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))