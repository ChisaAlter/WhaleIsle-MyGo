package dsh

import (
	"encoding/json"
	"fmt"
)

// MuxPath is the WebSocket route for every Typert Remote stream.
const MuxPath = "/api/remote.mux"

// Stream frame discriminators (JSON text messages on the mux socket).
const (
	FrameOpen   = "open"
	FrameItem   = "item"
	FrameEnd    = "end"
	FrameCancel = "cancel"
	FrameError  = "error"
)

// EventsEndpoint is the Gateway-internal stream for Cordis events.
const EventsEndpoint = "$events"

// EventsResultPath is the HTTP RPC that answers one waterfall.
const EventsResultPath = "/api/$events/result"

// EventsStreamPayload is the empty open payload for $events.
var EventsStreamPayload = json.RawMessage(`{"args":{}}`)

// ClientFrame is one browser→Host mux message.
type ClientFrame struct {
	Type     string          `json:"type"`
	StreamID string          `json:"streamId"`
	Endpoint string          `json:"endpoint,omitempty"`
	Payload  json.RawMessage `json:"payload,omitempty"`
	Value    json.RawMessage `json:"value,omitempty"`
}

// ServerFrame is one Host→browser mux message.
type ServerFrame struct {
	Type     string          `json:"type"`
	StreamID string          `json:"streamId"`
	Value    json.RawMessage `json:"value,omitempty"`
	Error    *StreamFailure  `json:"error,omitempty"`
}

// StreamFailure is the carrier-safe error on a server error frame.
type StreamFailure struct {
	Code    string          `json:"code"`
	Message string          `json:"message"`
	Details json.RawMessage `json:"details"`
}

func (e *StreamFailure) Error() string {
	if e == nil {
		return "dsh mux: stream failure"
	}
	if e.Code != "" {
		return e.Code + ": " + e.Message
	}
	return e.Message
}

func (f ClientFrame) Validate() error {
	switch f.Type {
	case FrameOpen:
		if f.StreamID == "" || f.Endpoint == "" {
			return fmt.Errorf("dsh mux: open requires streamId and endpoint")
		}
		return nil
	case FrameItem:
		if f.StreamID == "" {
			return fmt.Errorf("dsh mux: item requires streamId")
		}
		return nil
	case FrameEnd, FrameCancel:
		if f.StreamID == "" {
			return fmt.Errorf("dsh mux: %s requires streamId", f.Type)
		}
		return nil
	default:
		return fmt.Errorf("dsh mux: unknown client frame type %q", f.Type)
	}
}

func (f ServerFrame) Validate() error {
	switch f.Type {
	case FrameItem:
		if f.StreamID == "" {
			return fmt.Errorf("dsh mux: item requires streamId")
		}
		return nil
	case FrameEnd:
		if f.StreamID == "" {
			return fmt.Errorf("dsh mux: end requires streamId")
		}
		return nil
	case FrameError:
		if f.StreamID == "" || f.Error == nil {
			return fmt.Errorf("dsh mux: error requires streamId and error")
		}
		return nil
	default:
		return fmt.Errorf("dsh mux: unknown server frame type %q", f.Type)
	}
}

// Event downlink item discriminators inside $events stream values.
const (
	EventReady     = "ready"
	EventEmit      = "emit"
	EventWaterfall = "waterfall"
	EventCancel    = "cancel"
)

// EventsReady is the first $events item for one Client generation.
type EventsReady struct {
	Type     string `json:"type"`
	ClientID string `json:"clientId"`
	Host     struct {
		Home string `json:"home"`
	} `json:"host"`
}

// EventsEmit is a Host notification.
type EventsEmit struct {
	Type  string            `json:"type"`
	Event string            `json:"event"`
	Args  []json.RawMessage `json:"args"`
}

// EventsWaterfall is a pending Agent-scoped Host request.
type EventsWaterfall struct {
	Type    string          `json:"type"`
	Event   string          `json:"event"`
	EventID string          `json:"eventId"`
	AgentID string          `json:"agentId"`
	Request json.RawMessage `json:"request"`
}

// EventsCancel cancels a previously delivered waterfall.
type EventsCancel struct {
	Type    string `json:"type"`
	EventID string `json:"eventId"`
}

// EventResult is POST /api/$events/result.
type EventResult struct {
	ClientID string             `json:"clientId"`
	EventID  string             `json:"eventId"`
	Outcome  EventResultOutcome `json:"outcome"`
}

// EventResultOutcome is one of next / result / rejected.
type EventResultOutcome struct {
	Delivery *EventDelivery  `json:"-"`
	Kind     string          `json:"kind"`
	Value    json.RawMessage `json:"value,omitempty"`
	Error    *EventRejection `json:"error,omitempty"`
}

// EventRejection is a Client listener failure.
type EventRejection struct {
	Name    string          `json:"name"`
	Message string          `json:"message"`
	Code    string          `json:"code,omitempty"`
	Details json.RawMessage `json:"details,omitempty"`
}

// EventResultNext builds an unhandled-waterfall continuation.
func EventResultNext(clientID, eventID string) EventResult {
	return EventResult{
		ClientID: clientID,
		EventID:  eventID,
		Outcome:  EventResultOutcome{Kind: "next"},
	}
}
