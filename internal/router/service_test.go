package router

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/redis/go-redis/v9"
)

type fakeRedisGetter struct {
	value string
	err   error
}

func (f fakeRedisGetter) Get(_ context.Context, _ string) *redis.StringCmd {
	return redis.NewStringResult(f.value, f.err)
}

func TestRedisLookupBehaviors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		getter  fakeRedisGetter
		wantErr error
	}{
		{name: "found", getter: fakeRedisGetter{value: "gw-1"}},
		{name: "offline nil", getter: fakeRedisGetter{err: redis.Nil}, wantErr: ErrOffline},
		{name: "offline empty", getter: fakeRedisGetter{value: ""}, wantErr: ErrOffline},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			lookup := NewRedisLookup(tc.getter)
			_, err := lookup.GatewayInstanceID(context.Background(), "u1")
			if tc.wantErr == nil && err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if tc.wantErr != nil && !errors.Is(err, tc.wantErr) {
				t.Fatalf("expected %v got %v", tc.wantErr, err)
			}
		})
	}
}

type fakeLookup struct {
	instanceID string
	err        error
}

func (f fakeLookup) GatewayInstanceID(context.Context, string) (string, error) {
	return f.instanceID, f.err
}

type capturedPublish struct {
	subject string
	data    []byte
	err     error
}

func (p *capturedPublish) Publish(subject string, data []byte) error {
	p.subject = subject
	p.data = append([]byte(nil), data...)
	return p.err
}

func TestServiceRoutePublishesEnvelope(t *testing.T) {
	t.Parallel()
	publisher := &capturedPublish{}
	svc := NewService(fakeLookup{instanceID: "gw-2"}, publisher, false)

	instanceID, err := svc.Route(context.Background(), "u22", json.RawMessage(`{"kind":"chat"}`))
	if err != nil {
		t.Fatalf("Route() error = %v", err)
	}
	if instanceID != "gw-2" {
		t.Fatalf("expected gw-2 got %q", instanceID)
	}
	if publisher.subject != SendToUserSubject {
		t.Fatalf("expected subject %q got %q", SendToUserSubject, publisher.subject)
	}

	var envelope Envelope
	if err := json.Unmarshal(publisher.data, &envelope); err != nil {
		t.Fatalf("unmarshal publish payload: %v", err)
	}
	if envelope.UserID != "u22" || envelope.GatewayInstanceID != "gw-2" {
		t.Fatalf("unexpected envelope %+v", envelope)
	}
}

func TestServiceRouteTransientPublishFailure(t *testing.T) {
	t.Parallel()
	svc := NewService(fakeLookup{instanceID: "gw-7"}, &capturedPublish{err: errors.New("nats timeout")}, true)
	if _, err := svc.Route(context.Background(), "u22", json.RawMessage(`{"kind":"chat"}`)); err == nil {
		t.Fatal("expected publish error")
	}
}
