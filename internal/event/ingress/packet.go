package ingress

import (
	"encoding/json"
	"time"
)

type WebhookPacket struct {
	DeliveryID string              `json:"deliveryID"`
	EventName  string              `json:"eventName"`
	Action     string              `json:"action,omitempty"`
	Org        string              `json:"org,omitempty"`
	Repo       string              `json:"repo,omitempty"`
	Number     int                 `json:"number,omitempty"`
	SHA        string              `json:"sha,omitempty"`
	ReceivedAt time.Time           `json:"receivedAt"`
	Headers    map[string][]string `json:"headers,omitempty"`
	Payload    json.RawMessage     `json:"payload"`
}

type EnqueueResult struct {
	Duplicate     bool
	StreamEntryID string
}
