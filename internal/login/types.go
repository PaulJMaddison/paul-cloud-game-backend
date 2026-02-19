package login

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrInvalidUsername = errors.New("username must be between 3 and 64 characters")
	ErrInvalidPassword = errors.New("password must be between 8 and 128 characters")
)

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (r LoginRequest) Validate() error {
	r.Username = strings.TrimSpace(r.Username)
	if len(r.Username) < 3 || len(r.Username) > 64 {
		return ErrInvalidUsername
	}
	if len(r.Password) < 8 || len(r.Password) > 128 {
		return ErrInvalidPassword
	}
	return nil
}

type UserProfile struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	CreatedAt time.Time `json:"created_at"`
}

type LoginResponse struct {
	Token string      `json:"token"`
	User  UserProfile `json:"user"`
}

type MeResponse struct {
	User UserProfile `json:"user"`
}
