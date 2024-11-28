package broker
import (
    "time"

    "github.com/MuhammadYossry/AgentNexus/internal/task/types"
)

type Topic struct {
    Name            string
    Subscribers     map[string]*AIAgent
    MessageHandler  func(*Message) error
}

type Message struct {
    ID              string
    Topic           string
    Payload         interface{}
    Metadata        map[string]string
    Timestamp       time.Time
}