package gateway

import (
	"bufio"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	opcodeText  = 0x1
	opcodeClose = 0x8
	opcodePing  = 0x9
	opcodePong  = 0xA
)

var (
	errNotWebSocketUpgrade = errors.New("not a websocket upgrade request")
	errMaskedServerFrame   = errors.New("masked server frame")
)

type wsConn struct {
	netConn net.Conn
	br      *bufio.Reader
	mu      sync.Mutex
}

func upgradeWebSocket(w http.ResponseWriter, r *http.Request) (*wsConn, error) {
	if !headerContains(r.Header, "Connection", "upgrade") || !headerContains(r.Header, "Upgrade", "websocket") {
		return nil, errNotWebSocketUpgrade
	}
	if r.Header.Get("Sec-WebSocket-Version") != "13" {
		return nil, fmt.Errorf("unsupported websocket version")
	}
	key := r.Header.Get("Sec-WebSocket-Key")
	if key == "" {
		return nil, fmt.Errorf("missing sec-websocket-key")
	}

	h, ok := w.(http.Hijacker)
	if !ok {
		return nil, fmt.Errorf("http server does not support hijacking")
	}
	netConn, rw, err := h.Hijack()
	if err != nil {
		return nil, err
	}

	accept := websocketAccept(key)
	resp := "HTTP/1.1 101 Switching Protocols\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n" +
		"Sec-WebSocket-Accept: " + accept + "\r\n\r\n"
	if _, err := rw.WriteString(resp); err != nil {
		_ = netConn.Close()
		return nil, err
	}
	if err := rw.Flush(); err != nil {
		_ = netConn.Close()
		return nil, err
	}
	return &wsConn{netConn: netConn, br: rw.Reader}, nil
}

func websocketAccept(key string) string {
	h := sha1.New()
	_, _ = io.WriteString(h, strings.TrimSpace(key)+"258EAFA5-E914-47DA-95CA-C5AB0DC85B11")
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

func headerContains(h http.Header, key, want string) bool {
	for _, v := range h.Values(key) {
		for _, part := range strings.Split(v, ",") {
			if strings.EqualFold(strings.TrimSpace(part), want) {
				return true
			}
		}
	}
	return false
}

func (c *wsConn) Close() error {
	return c.netConn.Close()
}

func (c *wsConn) SetReadDeadline(t time.Time) error {
	return c.netConn.SetReadDeadline(t)
}

func (c *wsConn) SetWriteDeadline(t time.Time) error {
	return c.netConn.SetWriteDeadline(t)
}

func (c *wsConn) WriteText(payload []byte) error {
	return c.writeFrame(opcodeText, payload)
}

func (c *wsConn) WritePing(payload []byte) error {
	return c.writeFrame(opcodePing, payload)
}

func (c *wsConn) writeFrame(op byte, payload []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	head := make([]byte, 2)
	head[0] = 0x80 | op
	n := len(payload)
	switch {
	case n <= 125:
		head[1] = byte(n)
	case n <= 65535:
		head[1] = 126
		ext := make([]byte, 2)
		binary.BigEndian.PutUint16(ext, uint16(n))
		head = append(head, ext...)
	default:
		head[1] = 127
		ext := make([]byte, 8)
		binary.BigEndian.PutUint64(ext, uint64(n))
		head = append(head, ext...)
	}

	if _, err := c.netConn.Write(head); err != nil {
		return err
	}
	if n > 0 {
		_, err := c.netConn.Write(payload)
		return err
	}
	return nil
}

func (c *wsConn) ReadFrame() (byte, []byte, error) {
	hdr := make([]byte, 2)
	if _, err := io.ReadFull(c.br, hdr); err != nil {
		return 0, nil, err
	}

	opcode := hdr[0] & 0x0F
	masked := hdr[1]&0x80 != 0
	if !masked {
		return 0, nil, errMaskedServerFrame
	}

	payloadLen := int(hdr[1] & 0x7F)
	switch payloadLen {
	case 126:
		ext := make([]byte, 2)
		if _, err := io.ReadFull(c.br, ext); err != nil {
			return 0, nil, err
		}
		payloadLen = int(binary.BigEndian.Uint16(ext))
	case 127:
		ext := make([]byte, 8)
		if _, err := io.ReadFull(c.br, ext); err != nil {
			return 0, nil, err
		}
		payloadLen64 := binary.BigEndian.Uint64(ext)
		if payloadLen64 > 1<<31-1 {
			return 0, nil, fmt.Errorf("payload too large")
		}
		payloadLen = int(payloadLen64)
	}

	maskKey := make([]byte, 4)
	if _, err := io.ReadFull(c.br, maskKey); err != nil {
		return 0, nil, err
	}

	payload := make([]byte, payloadLen)
	if _, err := io.ReadFull(c.br, payload); err != nil {
		return 0, nil, err
	}
	for i := 0; i < payloadLen; i++ {
		payload[i] ^= maskKey[i%4]
	}
	return opcode, payload, nil
}
