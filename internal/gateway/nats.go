package gateway

import (
	"encoding/json"
	"errors"

	"github.com/nats-io/nats.go"
	"github.com/paul-cloud-game-backend/paul-cloud-game-backend/internal/contracts"
	"github.com/rs/zerolog"
)

type natsInbound struct {
	TargetUserID string          `json:"target_user_id"`
	Message      json.RawMessage `json:"message"`
}

func SubscribeSendToUser(nc *nats.Conn, logger zerolog.Logger, sender *userSender) (*nats.Subscription, error) {
	return nc.Subscribe(contracts.SubjectGatewaySendToUser, func(msg *nats.Msg) {
		userID, payload, err := decodeSendToUser(msg.Data)
		if err != nil {
			logger.Warn().Err(err).Msg("invalid nats send_to_user payload")
			return
		}
		if err := sender.SendToUser(userID, payload); err != nil {
			if err == ErrUserNotConnected {
				logger.Debug().Str("user_id", userID).Msg("user not connected on this gateway instance")
				return
			}
			logger.Warn().Err(err).Str("user_id", userID).Msg("failed to send nats message to user")
		}
	})
}

func decodeSendToUser(data []byte) (string, json.RawMessage, error) {
	var env contracts.Envelope
	if err := json.Unmarshal(data, &env); err == nil && env.Type == contracts.EventGatewaySendToUser {
		var payload contracts.GatewaySendToUserV1
		if err := json.Unmarshal(env.Payload, &payload); err != nil {
			return "", nil, err
		}
		if payload.TargetUserID == "" || len(payload.Message) == 0 {
			return "", nil, ErrInvalidMessage
		}
		return payload.TargetUserID, payload.Message, nil
	}

	var payload natsInbound
	if err := json.Unmarshal(data, &payload); err != nil {
		return "", nil, err
	}
	if payload.TargetUserID == "" || len(payload.Message) == 0 {
		return "", nil, ErrInvalidMessage
	}
	return payload.TargetUserID, payload.Message, nil
}

var ErrInvalidMessage = errors.New("invalid gateway send_to_user message")
