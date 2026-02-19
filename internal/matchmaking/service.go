package matchmaking

import (
	"context"
	"encoding/json"
	"time"

	"github.com/paul-cloud-game-backend/paul-cloud-game-backend/internal/contracts"
)

type Publisher interface {
	Publish(subject string, data []byte) error
}

type Service struct {
	queue     Queue
	publisher Publisher
	now       func() time.Time
	newID     func() (string, error)
}

type MatchResult struct {
	UserA   string
	UserB   string
	MatchID string
}

func BuildMatch(ids []string, matchID string) (MatchResult, bool) {
	if len(ids) < 2 {
		return MatchResult{}, false
	}
	return MatchResult{UserA: ids[0], UserB: ids[1], MatchID: matchID}, true
}

func NewService(queue Queue, publisher Publisher) *Service {
	return &Service{queue: queue, publisher: publisher, now: func() time.Time { return time.Now().UTC() }, newID: newUUID}
}

func (s *Service) Enqueue(ctx context.Context, userID, correlationID string) error {
	if err := s.queue.Enqueue(ctx, userID); err != nil {
		return err
	}
	eventID, err := s.newID()
	if err != nil {
		return err
	}
	payload := contracts.MatchmakingEnqueuedV1{TicketID: eventID, Queue: "default"}
	raw, err := contracts.MarshalV1(eventID, contracts.EventMatchmakingEnqueued, s.now(), correlationID, &userID, payload)
	if err != nil {
		return err
	}
	return s.publisher.Publish(contracts.SubjectMatchmakingQueued, raw)
}

func (s *Service) ProcessOnce(ctx context.Context) error {
	ids, err := s.queue.DequeuePair(ctx)
	if err != nil || len(ids) < 2 {
		return err
	}

	matchID, err := s.newID()
	if err != nil {
		return err
	}
	res, ok := BuildMatch(ids, matchID)
	if !ok {
		return nil
	}
	corrID, err := s.newID()
	if err != nil {
		return err
	}

	if err := s.publishMatched(corrID, res); err != nil {
		return err
	}
	if err := s.publishUserMessage(corrID, res.UserA, res.UserB, res.MatchID); err != nil {
		return err
	}
	return s.publishUserMessage(corrID, res.UserB, res.UserA, res.MatchID)
}

func (s *Service) Run(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = s.ProcessOnce(ctx)
		}
	}
}

func (s *Service) publishMatched(correlationID string, result MatchResult) error {
	eventID, err := s.newID()
	if err != nil {
		return err
	}
	payload := contracts.MatchmakingMatchedV1{MatchID: result.MatchID, UserIDs: []string{result.UserA, result.UserB}}
	raw, err := contracts.MarshalV1(eventID, contracts.EventMatchmakingMatched, s.now(), correlationID, nil, payload)
	if err != nil {
		return err
	}
	return s.publisher.Publish(contracts.SubjectMatchmakingMatch, raw)
}

func (s *Service) publishUserMessage(correlationID, targetUserID, otherUserID, matchID string) error {
	eventID, err := s.newID()
	if err != nil {
		return err
	}
	message, err := json.Marshal(map[string]string{"type": "match_found", "match_id": matchID, "other_user_id": otherUserID})
	if err != nil {
		return err
	}
	payload := contracts.GatewaySendToUserV1{TargetUserID: targetUserID, Message: message}
	raw, err := contracts.MarshalV1(eventID, contracts.EventGatewaySendToUser, s.now(), correlationID, &targetUserID, payload)
	if err != nil {
		return err
	}
	return s.publisher.Publish(contracts.SubjectGatewaySendToUser, raw)
}
