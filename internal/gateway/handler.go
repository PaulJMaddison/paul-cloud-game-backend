package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/paul-cloud-game-backend/paul-cloud-game-backend/pkg/apierror"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
)

const (
	defaultPresenceTTL      = 60 * time.Second
	defaultPresenceInterval = 20 * time.Second
	writeWait               = 10 * time.Second
	pongWait                = 70 * time.Second
	pingPeriod              = 25 * time.Second
)

type TokenParser interface {
	ParseToken(token string) (string, string, error)
}

type SendRequest struct {
	UserID  string          `json:"user_id"`
	Message json.RawMessage `json:"message"`
}

type userSender struct {
	instanceID string
	logger     zerolog.Logger
	redis      *redis.Client
	parser     TokenParser

	presenceTTL      time.Duration
	presenceInterval time.Duration

	mu    sync.RWMutex
	conns map[string]*clientConn
}

type clientConn struct {
	conn *wsConn
	mu   sync.Mutex
}

func NewSender(instanceID string, logger zerolog.Logger, redisClient *redis.Client, parser TokenParser) *userSender {
	return &userSender{instanceID: instanceID, logger: logger, redis: redisClient, parser: parser, presenceTTL: defaultPresenceTTL, presenceInterval: defaultPresenceInterval, conns: make(map[string]*clientConn)}
}

func (s *userSender) Register(mux *http.ServeMux) {
	mux.HandleFunc("/v1/ws", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			apierror.Write(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}

		token := strings.TrimSpace(r.URL.Query().Get("token"))
		if token == "" {
			apierror.Write(w, http.StatusUnauthorized, "unauthorized", "missing token")
			return
		}
		userID, _, err := s.parser.ParseToken(token)
		if err != nil {
			apierror.Write(w, http.StatusUnauthorized, "unauthorized", "invalid token")
			return
		}

		conn, err := upgradeWebSocket(w, r)
		if err != nil {
			s.logger.Error().Err(err).Str("user_id", userID).Msg("upgrade websocket")
			return
		}
		s.handleConnection(r.Context(), userID, conn)
	})

	mux.HandleFunc("/v1/send", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			apierror.Write(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}
		var req SendRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			apierror.Write(w, http.StatusBadRequest, "invalid_json", "invalid json")
			return
		}
		if req.UserID == "" || len(req.Message) == 0 {
			apierror.Write(w, http.StatusBadRequest, "validation_failed", "user_id and message are required")
			return
		}
		if err := s.SendToUser(req.UserID, req.Message); err != nil {
			if errors.Is(err, ErrUserNotConnected) {
				apierror.Write(w, http.StatusNotFound, "not_connected", err.Error())
				return
			}
			apierror.Write(w, http.StatusInternalServerError, "internal_error", "failed to send message")
			return
		}
		w.WriteHeader(http.StatusAccepted)
	})
}

func (s *userSender) handleConnection(reqCtx context.Context, userID string, conn *wsConn) {
	cc := &clientConn{conn: conn}
	s.mu.Lock()
	s.conns[userID] = cc
	s.mu.Unlock()

	ctx, cancel := context.WithCancel(reqCtx)
	defer cancel()
	defer func() {
		s.mu.Lock()
		delete(s.conns, userID)
		s.mu.Unlock()
		_ = s.redis.Del(context.Background(), presenceKey(userID)).Err()
		_ = conn.Close()
	}()

	if err := s.refreshPresence(ctx, userID); err != nil {
		s.logger.Warn().Err(err).Str("user_id", userID).Msg("failed to set initial redis presence")
	}
	go s.presenceLoop(ctx, userID)

	_ = conn.SetReadDeadline(time.Now().Add(pongWait))

	pingDone := make(chan struct{})
	go func() {
		defer close(pingDone)
		ticker := time.NewTicker(pingPeriod)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				cc.mu.Lock()
				_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
				err := conn.WritePing([]byte("ping"))
				cc.mu.Unlock()
				if err != nil {
					cancel()
					return
				}
			}
		}
	}()

	for {
		opcode, payload, err := conn.ReadFrame()
		if err != nil {
			cancel()
			break
		}
		switch opcode {
		case opcodeClose:
			cancel()
		case opcodePong:
			_ = conn.SetReadDeadline(time.Now().Add(pongWait))
		case opcodePing:
			cc.mu.Lock()
			_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
			_ = conn.writeFrame(opcodePong, payload)
			cc.mu.Unlock()
		}
		if ctx.Err() != nil {
			break
		}
	}
	<-pingDone
}

func (s *userSender) presenceLoop(ctx context.Context, userID string) {
	ticker := time.NewTicker(s.presenceInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.refreshPresence(ctx, userID); err != nil {
				s.logger.Warn().Err(err).Str("user_id", userID).Msg("failed to refresh redis presence")
			}
		}
	}
}

func (s *userSender) refreshPresence(ctx context.Context, userID string) error {
	return s.redis.Set(ctx, presenceKey(userID), s.instanceID, s.presenceTTL).Err()
}

var ErrUserNotConnected = errors.New("user not connected")

func (s *userSender) SendToUser(userID string, message json.RawMessage) error {
	s.mu.RLock()
	cc, ok := s.conns[userID]
	s.mu.RUnlock()
	if !ok {
		return ErrUserNotConnected
	}

	cc.mu.Lock()
	defer cc.mu.Unlock()
	_ = cc.conn.SetWriteDeadline(time.Now().Add(writeWait))
	return cc.conn.WriteText(message)
}

func presenceKey(userID string) string { return "pcgb:gateway:user:" + userID }
