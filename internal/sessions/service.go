package sessions

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/paul-cloud-game-backend/paul-cloud-game-backend/internal/contracts"
)

var ErrForbidden = errors.New("forbidden")

type TokenParser interface {
	ParseToken(token string) (string, string, error)
}

type Service struct {
	repo   Repository
	auth   TokenParser
	nc     *nats.Conn
	server ServerAllocation
}

func NewService(repo Repository, auth TokenParser, nc *nats.Conn) *Service {
	return &Service{repo: repo, auth: auth, nc: nc, server: ServerAllocation{IP: "127.0.0.1", Port: 7777, Region: "local"}}
}

func (s *Service) ParseToken(token string) (string, error) {
	userID, _, err := s.auth.ParseToken(token)
	return userID, err
}

func (s *Service) CreateSessionForUser(ctx context.Context, userID, correlationID string) (Session, error) {
	session, err := s.repo.CreateSession(ctx, userID, "created", []string{userID})
	if err != nil {
		return Session{}, err
	}
	if err := s.publishSessionCreated(correlationID, userID, session.ID); err != nil {
		return Session{}, err
	}
	return session, nil
}

func (s *Service) AssignServer(ctx context.Context, userID, sessionID, correlationID string) (AssignServerResponse, error) {
	isMember, err := s.repo.IsMember(ctx, sessionID, userID)
	if err != nil {
		return AssignServerResponse{}, err
	}
	if !isMember {
		return AssignServerResponse{}, ErrForbidden
	}
	if err := s.publishSessionAssigned(correlationID, userID, sessionID); err != nil {
		return AssignServerResponse{}, err
	}
	return AssignServerResponse{Server: s.server}, nil
}

func (s *Service) HandleMatchedEvent(msg *nats.Msg) {
	env, err := contracts.UnmarshalEnvelope(msg.Data)
	if err != nil || env.Type != contracts.EventMatchmakingMatched {
		return
	}
	var payload contracts.MatchmakingMatchedV1
	if err := json.Unmarshal(env.Payload, &payload); err != nil {
		return
	}
	members := payload.UserIDs
	if len(members) == 0 {
		members = payload.SessionIDs
	}
	if len(members) == 0 {
		return
	}
	owner := members[0]
	session, err := s.repo.CreateSession(context.Background(), owner, "created", members)
	if err != nil {
		return
	}
	_ = s.publishSessionCreated(env.CorrelationID, owner, session.ID)
}

func (s *Service) publishSessionCreated(correlationID, userID, sessionID string) error {
	if s.nc == nil {
		return nil
	}
	eventID, err := newUUID()
	if err != nil {
		return err
	}
	payload := contracts.SessionCreatedV1{SessionID: sessionID}
	raw, err := contracts.MarshalV1(eventID, contracts.EventSessionCreated, time.Now().UTC(), correlationID, &userID, payload)
	if err != nil {
		return err
	}
	msg := nats.NewMsg(contracts.SubjectSessionCreated)
	msg.Data = raw
	msg.Header.Set("correlation_id", correlationID)
	msg.Header.Set("content-type", "application/json")
	return s.nc.PublishMsg(msg)
}

func (s *Service) publishSessionAssigned(correlationID, userID, sessionID string) error {
	if s.nc == nil {
		return nil
	}
	eventID, err := newUUID()
	if err != nil {
		return err
	}
	payload := contracts.SessionAssignedServerV1{SessionID: sessionID, ServerID: "local-7777"}
	raw, err := contracts.MarshalV1(eventID, contracts.EventSessionAssigned, time.Now().UTC(), correlationID, &userID, payload)
	if err != nil {
		return err
	}
	msg := nats.NewMsg(contracts.SubjectSessionAssigned)
	msg.Data = raw
	msg.Header.Set("correlation_id", correlationID)
	msg.Header.Set("content-type", "application/json")
	return s.nc.PublishMsg(msg)
}
