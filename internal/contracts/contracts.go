package contracts

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// EventType identifies the semantic event kind.
type EventType string

const (
	EventUserLoggedIn        EventType = "user.logged_in"
	EventSessionCreated      EventType = "session.created"
	EventSessionAssigned     EventType = "session.assigned_server"
	EventMatchmakingEnqueued EventType = "matchmaking.enqueued"
	EventMatchmakingMatched  EventType = "matchmaking.matched"
	EventGatewaySendToUser   EventType = "gateway.send_to_user"
)

var validEventTypes = map[EventType]struct{}{
	EventUserLoggedIn:        {},
	EventSessionCreated:      {},
	EventSessionAssigned:     {},
	EventMatchmakingEnqueued: {},
	EventMatchmakingMatched:  {},
	EventGatewaySendToUser:   {},
}

// Envelope is the JSON-serializable event envelope shared across services.
type Envelope struct {
	ID            string          `json:"id"`
	Type          EventType       `json:"type"`
	TS            time.Time       `json:"ts"`
	CorrelationID string          `json:"correlation_id"`
	UserID        *string         `json:"user_id,omitempty"`
	Payload       json.RawMessage `json:"payload"`
}

var ErrInvalidEventType = errors.New("invalid event type")

// ValidateEventType verifies whether the provided event type is known.
func ValidateEventType(eventType EventType) error {
	if _, ok := validEventTypes[eventType]; !ok {
		return fmt.Errorf("%w: %s", ErrInvalidEventType, eventType)
	}
	return nil
}

// MarshalV1 marshals an envelope with a v1 payload struct.
func MarshalV1[T any](id string, eventType EventType, ts time.Time, correlationID string, userID *string, payload T) ([]byte, error) {
	if err := ValidateEventType(eventType); err != nil {
		return nil, err
	}

	payloadRaw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	env := Envelope{
		ID:            id,
		Type:          eventType,
		TS:            ts,
		CorrelationID: correlationID,
		UserID:        userID,
		Payload:       payloadRaw,
	}

	return json.Marshal(env)
}

// UnmarshalEnvelope unmarshals and validates an event envelope.
func UnmarshalEnvelope(data []byte) (Envelope, error) {
	var env Envelope
	if err := json.Unmarshal(data, &env); err != nil {
		return Envelope{}, err
	}
	if err := ValidateEventType(env.Type); err != nil {
		return Envelope{}, err
	}
	return env, nil
}

// V1 payload schemas.
type UserLoggedInV1 struct {
	AuthMethod string `json:"auth_method,omitempty"`
}

type SessionCreatedV1 struct {
	SessionID string `json:"session_id"`
}

type SessionAssignedServerV1 struct {
	SessionID string `json:"session_id"`
	ServerID  string `json:"server_id"`
}

type MatchmakingEnqueuedV1 struct {
	TicketID string `json:"ticket_id"`
	Queue    string `json:"queue"`
}

type MatchmakingMatchedV1 struct {
	MatchID    string   `json:"match_id"`
	SessionIDs []string `json:"session_ids,omitempty"`
	UserIDs    []string `json:"user_ids,omitempty"`
}

type GatewaySendToUserV1 struct {
	TargetUserID string          `json:"target_user_id"`
	Message      json.RawMessage `json:"message"`
}

// DecodeV1Payload decodes the payload into a v1 schema by event type.
func DecodeV1Payload(env Envelope) (any, error) {
	switch env.Type {
	case EventUserLoggedIn:
		var payload UserLoggedInV1
		return payload, json.Unmarshal(env.Payload, &payload)
	case EventSessionCreated:
		var payload SessionCreatedV1
		return payload, json.Unmarshal(env.Payload, &payload)
	case EventSessionAssigned:
		var payload SessionAssignedServerV1
		return payload, json.Unmarshal(env.Payload, &payload)
	case EventMatchmakingEnqueued:
		var payload MatchmakingEnqueuedV1
		return payload, json.Unmarshal(env.Payload, &payload)
	case EventMatchmakingMatched:
		var payload MatchmakingMatchedV1
		return payload, json.Unmarshal(env.Payload, &payload)
	case EventGatewaySendToUser:
		var payload GatewaySendToUserV1
		return payload, json.Unmarshal(env.Payload, &payload)
	default:
		return nil, fmt.Errorf("%w: %s", ErrInvalidEventType, env.Type)
	}
}

// NATS subject mapping.
const (
	SubjectUserLoggedIn      = "pcgb.user.logged_in"
	SubjectSessionCreated    = "pcgb.session.created"
	SubjectSessionAssigned   = "pcgb.session.assigned_server"
	SubjectMatchmakingQueued = "pcgb.mm.enqueued"
	SubjectMatchmakingMatch  = "pcgb.mm.matched"
	SubjectGatewaySendToUser = "pcgb.gateway.send_to_user"
)

// SubjectForType maps a contract event type to its NATS subject.
func SubjectForType(eventType EventType) (string, error) {
	switch eventType {
	case EventUserLoggedIn:
		return SubjectUserLoggedIn, nil
	case EventSessionCreated:
		return SubjectSessionCreated, nil
	case EventSessionAssigned:
		return SubjectSessionAssigned, nil
	case EventMatchmakingEnqueued:
		return SubjectMatchmakingQueued, nil
	case EventMatchmakingMatched:
		return SubjectMatchmakingMatch, nil
	case EventGatewaySendToUser:
		return SubjectGatewaySendToUser, nil
	default:
		return "", fmt.Errorf("%w: %s", ErrInvalidEventType, eventType)
	}
}
