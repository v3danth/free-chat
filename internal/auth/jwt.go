package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/v3danth/free-chat/internal/user"
)

type Claims struct {
	UserID   uint64        `json:"user_id"`
	UserType user.UserType `json:"user_type"`
	Username string        `json:"username"`
	jwt.RegisteredClaims
}

type JWTManager struct {
	secret string
}

func NewJWTManager(secret string) *JWTManager {
	return &JWTManager{secret: secret}
}

func (j *JWTManager) Generate(userID uint64, userType user.UserType, username string) (string, error) {
	claims := Claims{
		UserID:   userID,
		UserType: userType,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(j.secret))
}

func (j *JWTManager) Validate(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(
		tokenStr,
		&Claims{},
		func(token *jwt.Token) (interface{}, error) {
			return []byte(j.secret), nil
		},
	)
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token claims")
	}

	switch claims.UserType {
	case user.TypeGuest, user.TypeRegistered:
		// ok
	default:
		return nil, errors.New("invalid user type in token")
	}

	return claims, nil
}
