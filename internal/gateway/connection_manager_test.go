package gateway

import (
	"encoding/json"
	"io"
	"net"
	"sync"
	"testing"
)

func TestSendToUserOffline(t *testing.T) {
	t.Parallel()
	s := &userSender{conns: map[string]*clientConn{}}
	if err := s.SendToUser("missing", json.RawMessage(`{"x":1}`)); err != ErrUserNotConnected {
		t.Fatalf("expected ErrUserNotConnected, got %v", err)
	}
}

func TestSendToUserOnlineAndConcurrent(t *testing.T) {
	t.Parallel()
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	go io.Copy(io.Discard, server)
	ws := &wsConn{netConn: client}
	s := &userSender{conns: map[string]*clientConn{"u1": {conn: ws}}}

	const n = 20
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			_ = s.SendToUser("u1", json.RawMessage(`{"type":"ping"}`))
		}()
	}
	wg.Wait()
}
