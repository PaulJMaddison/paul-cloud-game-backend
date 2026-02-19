package login

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

var ErrInvalidToken = errors.New("invalid token")

type Authenticator struct {
	secret []byte
	ttl    time.Duration
}

type tokenClaims struct {
	Sub      string `json:"sub"`
	Username string `json:"username"`
	Iat      int64  `json:"iat"`
	Exp      int64  `json:"exp"`
}

func NewAuthenticator(secret string, ttl time.Duration) *Authenticator {
	return &Authenticator{secret: []byte(secret), ttl: ttl}
}

func (a *Authenticator) HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func (a *Authenticator) VerifyPassword(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

func (a *Authenticator) GenerateToken(userID, username string) (string, error) {
	header := map[string]string{"alg": "HS256", "typ": "JWT"}
	now := time.Now().UTC()
	claims := tokenClaims{Sub: userID, Username: username, Iat: now.Unix(), Exp: now.Add(a.ttl).Unix()}

	headerRaw, err := json.Marshal(header)
	if err != nil {
		return "", err
	}
	claimsRaw, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}

	head := base64.RawURLEncoding.EncodeToString(headerRaw)
	body := base64.RawURLEncoding.EncodeToString(claimsRaw)
	sig := a.sign(head + "." + body)
	return head + "." + body + "." + sig, nil
}

func (a *Authenticator) ParseToken(token string) (string, string, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return "", "", ErrInvalidToken
	}

	expected := a.sign(parts[0] + "." + parts[1])
	if !hmac.Equal([]byte(expected), []byte(parts[2])) {
		return "", "", ErrInvalidToken
	}

	claimsBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", "", ErrInvalidToken
	}
	var claims tokenClaims
	if err := json.Unmarshal(claimsBytes, &claims); err != nil {
		return "", "", ErrInvalidToken
	}
	if claims.Sub == "" || claims.Username == "" || claims.Exp < time.Now().UTC().Unix() {
		return "", "", ErrInvalidToken
	}
	return claims.Sub, claims.Username, nil
}

func (a *Authenticator) sign(payload string) string {
	h := hmac.New(sha256.New, a.secret)
	_, _ = h.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}
