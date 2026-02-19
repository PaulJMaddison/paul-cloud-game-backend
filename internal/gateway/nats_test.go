package gateway

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/paul-cloud-game-backend/paul-cloud-game-backend/internal/contracts"
)

func TestDecodeSendToUserRaw(t *testing.T) {
	data := []byte(`{"target_user_id":"u1","message":{"hello":"world"}}`)
	uid, payload, err := decodeSendToUser(data)
	if err != nil {
		t.Fatalf("decodeSendToUser error: %v", err)
	}
	if uid != "u1" {
		t.Fatalf("user id mismatch: got %q", uid)
	}
	if string(payload) != `{"hello":"world"}` {
		t.Fatalf("payload mismatch: %s", payload)
	}
}

func TestDecodeSendToUserEnvelope(t *testing.T) {
	raw, err := contracts.MarshalV1("id1", contracts.EventGatewaySendToUser, time.Now().UTC(), "corr", nil, contracts.GatewaySendToUserV1{
		TargetUserID: "u2",
		Message:      json.RawMessage(`{"type":"notice"}`),
	})
	if err != nil {
		t.Fatalf("marshal envelope: %v", err)
	}

	uid, payload, err := decodeSendToUser(raw)
	if err != nil {
		t.Fatalf("decodeSendToUser envelope error: %v", err)
	}
	if uid != "u2" {
		t.Fatalf("user id mismatch: got %q", uid)
	}
	if string(payload) != `{"type":"notice"}` {
		t.Fatalf("payload mismatch: %s", payload)
	}
}
