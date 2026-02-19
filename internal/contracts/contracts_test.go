package contracts

import (
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestMarshalUnmarshalAndDecodeV1(t *testing.T) {
	tests := []struct {
		name      string
		eventType EventType
		payload   any
	}{
		{name: "user logged in", eventType: EventUserLoggedIn, payload: UserLoggedInV1{AuthMethod: "steam"}},
		{name: "session created", eventType: EventSessionCreated, payload: SessionCreatedV1{SessionID: "s-1"}},
		{name: "session assigned", eventType: EventSessionAssigned, payload: SessionAssignedServerV1{SessionID: "s-1", ServerID: "srv-1"}},
		{name: "matchmaking enqueued", eventType: EventMatchmakingEnqueued, payload: MatchmakingEnqueuedV1{TicketID: "t-1", Queue: "ranked"}},
		{name: "matchmaking matched", eventType: EventMatchmakingMatched, payload: MatchmakingMatchedV1{MatchID: "m-1", SessionIDs: []string{"s-1", "s-2"}}},
		{name: "gateway send to user", eventType: EventGatewaySendToUser, payload: GatewaySendToUserV1{TargetUserID: "u-1", Message: json.RawMessage(`{"op":"notify"}`)}},
	}

	ts := time.Now().UTC().Round(time.Second)
	userID := "user-123"

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw, err := MarshalV1("evt-1", tt.eventType, ts, "corr-1", &userID, tt.payload)
			if err != nil {
				t.Fatalf("MarshalV1 error: %v", err)
			}

			env, err := UnmarshalEnvelope(raw)
			if err != nil {
				t.Fatalf("UnmarshalEnvelope error: %v", err)
			}

			decoded, err := DecodeV1Payload(env)
			if err != nil {
				t.Fatalf("DecodeV1Payload error: %v", err)
			}

			decodedRaw, _ := json.Marshal(decoded)
			expectedRaw, _ := json.Marshal(tt.payload)
			if string(decodedRaw) != string(expectedRaw) {
				t.Fatalf("decoded payload mismatch\n got: %s\nwant: %s", decodedRaw, expectedRaw)
			}
		})
	}
}

func TestDecodeV1BackwardCompatibleWithAdditionalFields(t *testing.T) {
	raw := []byte(`{
		"id":"evt-legacy",
		"type":"session.created",
		"ts":"2025-01-01T00:00:00Z",
		"correlation_id":"corr-legacy",
		"payload":{
			"session_id":"s-legacy",
			"new_field":"ignored"
		},
		"new_envelope_field":"ignored"
	}`)

	env, err := UnmarshalEnvelope(raw)
	if err != nil {
		t.Fatalf("UnmarshalEnvelope error: %v", err)
	}

	decoded, err := DecodeV1Payload(env)
	if err != nil {
		t.Fatalf("DecodeV1Payload error: %v", err)
	}

	payload, ok := decoded.(SessionCreatedV1)
	if !ok {
		t.Fatalf("unexpected payload type: %T", decoded)
	}

	if payload.SessionID != "s-legacy" {
		t.Fatalf("SessionID mismatch: got %s", payload.SessionID)
	}
}

func TestValidateEventType(t *testing.T) {
	err := ValidateEventType(EventType("unknown.event"))
	if !errors.Is(err, ErrInvalidEventType) {
		t.Fatalf("expected ErrInvalidEventType, got %v", err)
	}
}

func TestSubjectForType(t *testing.T) {
	subject, err := SubjectForType(EventMatchmakingMatched)
	if err != nil {
		t.Fatalf("SubjectForType error: %v", err)
	}
	if subject != "pcgb.mm.matched" {
		t.Fatalf("unexpected subject: %s", subject)
	}
}
