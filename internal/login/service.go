package login

import (
	"context"
	"errors"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/paul-cloud-game-backend/paul-cloud-game-backend/internal/contracts"
)

type Service struct {
	repo Repository
	auth *Authenticator
	nc   *nats.Conn
}

func NewService(repo Repository, auth *Authenticator, nc *nats.Conn) *Service {
	return &Service{repo: repo, auth: auth, nc: nc}
}

func (s *Service) Login(ctx context.Context, req LoginRequest, correlationID string) (LoginResponse, error) {
	if err := req.Validate(); err != nil {
		return LoginResponse{}, err
	}

	user, err := s.repo.GetByUsername(ctx, req.Username)
	if err != nil {
		if !errors.Is(err, ErrUserNotFound) {
			return LoginResponse{}, err
		}
		hash, err := s.auth.HashPassword(req.Password)
		if err != nil {
			return LoginResponse{}, err
		}
		user, err = s.repo.Create(ctx, req.Username, hash)
		if err != nil {
			return LoginResponse{}, err
		}
	} else if user.PasswordHash == "" {
		hash, err := s.auth.HashPassword(req.Password)
		if err != nil {
			return LoginResponse{}, err
		}
		if err := s.repo.UpdatePassword(ctx, user.ID, hash); err != nil {
			return LoginResponse{}, err
		}
	} else if err := s.auth.VerifyPassword(user.PasswordHash, req.Password); err != nil {
		return LoginResponse{}, ErrInvalidCredentials
	}

	token, err := s.auth.GenerateToken(user.ID, user.Username)
	if err != nil {
		return LoginResponse{}, err
	}

	if correlationID == "" {
		correlationID, err = newUUID()
		if err != nil {
			return LoginResponse{}, err
		}
	}
	if err := s.publishLoggedIn(correlationID, user); err != nil {
		return LoginResponse{}, err
	}

	return LoginResponse{Token: token, User: mapUser(user)}, nil
}

func (s *Service) Me(ctx context.Context, userID string) (UserProfile, error) {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return UserProfile{}, err
	}
	return mapUser(user), nil
}

func (s *Service) ParseToken(token string) (string, string, error) {
	return s.auth.ParseToken(token)
}

func (s *Service) publishLoggedIn(correlationID string, user User) error {
	eventID, err := newUUID()
	if err != nil {
		return err
	}
	payload := contracts.UserLoggedInV1{AuthMethod: "password"}
	raw, err := contracts.MarshalV1(eventID, contracts.EventUserLoggedIn, time.Now().UTC(), correlationID, &user.ID, payload)
	if err != nil {
		return err
	}
	msg := nats.NewMsg(contracts.SubjectUserLoggedIn)
	msg.Data = raw
	msg.Header.Set("correlation_id", correlationID)
	msg.Header.Set("content-type", "application/json")
	return s.nc.PublishMsg(msg)
}

func mapUser(user User) UserProfile {
	return UserProfile{ID: user.ID, Username: user.Username, CreatedAt: user.CreatedAt}
}
