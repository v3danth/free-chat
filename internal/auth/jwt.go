package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/v3danth/free-chat/internal/apperr"
	"github.com/v3danth/free-chat/internal/user"
)

const tokenTTL = 24 * time.Hour

var ErrInvalidToken = apperr.New(apperr.Unauthorized, "invalid or expired token")

// claims is the token's wire format; it never leaves this file. Role and
// profile are deliberately absent: they are read fresh from the database.
type claims struct {
	UserID  uint64 `json:"uid"`
	Version uint32 `json:"ver"`
	jwt.RegisteredClaims
}

type Tokens struct {
	secret []byte
}

func NewTokens(secret string) Tokens {
	return Tokens{secret: []byte(secret)}
}

func (t Tokens) Issue(u user.User) (string, error) {
	now := time.Now()
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims{
		UserID:  u.ID,
		Version: u.TokenVersion,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(tokenTTL)),
		},
	}).SignedString(t.secret)
}

// verify returns the user id and token version a valid token carries.
func (t Tokens) verify(raw string) (uint64, uint32, error) {
	var c claims
	_, err := jwt.ParseWithClaims(raw, &c,
		func(*jwt.Token) (any, error) { return t.secret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithExpirationRequired(),
	)
	if err != nil || c.UserID == 0 {
		return 0, 0, ErrInvalidToken
	}
	return c.UserID, c.Version, nil
}
