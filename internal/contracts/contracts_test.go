package contracts

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestGoldenVectors(t *testing.T) {
	t.Parallel()
	files, err := filepath.Glob("testdata/*.json")
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("expected golden vectors")
	}
	for _, file := range files {
		file := file
		t.Run(filepath.Base(file), func(t *testing.T) {
			t.Parallel()
			raw, err := os.ReadFile(file)
			if err != nil {
				t.Fatalf("read %s: %v", file, err)
			}
			env, err := UnmarshalEnvelope(raw)
			if err != nil {
				t.Fatalf("unmarshal envelope: %v", err)
			}
			if _, err := DecodeV1Payload(env); err != nil {
				t.Fatalf("decode payload: %v", err)
			}
		})
	}
}

func TestMarshalRoundTripAllV1Types(t *testing.T) {
	t.Parallel()
	ts := time.Now().UTC().Round(time.Second)
	userID := "user-123"
	tests := []struct {
		name    string
		typ     EventType
		payload any
	}{
		{"user", EventUserLoggedIn, UserLoggedInV1{AuthMethod: "steam"}},
		{"session created", EventSessionCreated, SessionCreatedV1{SessionID: "s-1"}},
		{"session assigned", EventSessionAssigned, SessionAssignedServerV1{SessionID: "s-1", ServerID: "srv-1"}},
		{"queue", EventMatchmakingEnqueued, MatchmakingEnqueuedV1{TicketID: "t-1", Queue: "ranked"}},
		{"matched", EventMatchmakingMatched, MatchmakingMatchedV1{MatchID: "m-1", UserIDs: []string{"u-1", "u-2"}}},
		{"send", EventGatewaySendToUser, GatewaySendToUserV1{TargetUserID: "u-1", Message: json.RawMessage(`{"op":"notify"}`)}},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			raw, err := MarshalV1("evt-1", tt.typ, ts, "corr-1", &userID, tt.payload)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			env, err := UnmarshalEnvelope(raw)
			if err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			dec, err := DecodeV1Payload(env)
			if err != nil {
				t.Fatalf("decode: %v", err)
			}
			got, _ := json.Marshal(dec)
			want, _ := json.Marshal(tt.payload)
			if string(got) != string(want) {
				t.Fatalf("mismatch got=%s want=%s", got, want)
			}
		})
	}
}
