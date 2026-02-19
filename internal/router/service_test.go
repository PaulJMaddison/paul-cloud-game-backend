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
	cmd := redis.NewStringResult(f.value, f.err)
	return cmd
}

func TestRedisLookupGatewayInstanceIDFound(t *testing.T) {
	lookup := NewRedisLookup(fakeRedisGetter{value: "gw-1"})

	instanceID, err := lookup.GatewayInstanceID(context.Background(), "u1")
	if err != nil {
		t.Fatalf("GatewayInstanceID() error = %v", err)
	}
	if instanceID != "gw-1" {
		t.Fatalf("expected gw-1, got %q", instanceID)
	}
}

func TestRedisLookupGatewayInstanceIDOffline(t *testing.T) {
	lookup := NewRedisLookup(fakeRedisGetter{err: redis.Nil})

	_, err := lookup.GatewayInstanceID(context.Background(), "missing")
	if !errors.Is(err, ErrOffline) {
		t.Fatalf("expected ErrOffline, got %v", err)
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

func TestServiceRoutePartitionedSubject(t *testing.T) {
	publisher := &capturedPublish{}
	svc := NewService(fakeLookup{instanceID: "gw-7"}, publisher, true)

	_, err := svc.Route(context.Background(), "u22", json.RawMessage(`{"kind":"chat"}`))
	if err != nil {
		t.Fatalf("Route() error = %v", err)
	}

	want := SendToUserSubject + ".gw-7"
	if publisher.subject != want {
		t.Fatalf("expected subject %q got %q", want, publisher.subject)
	}
}
