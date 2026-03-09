package jwt

import (
	"fmt"
	"strconv"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type User struct {
	ID uint64
}

type Token interface {
	Create(user *User) (string, error)

	Verify(token string, user *User) error
}

type userClaims struct {
	jwtlib.RegisteredClaims

	UserID uint64 `json:"user_id"`
}

type jwtToken struct {
	secret   string
	duration time.Duration
}

func NewToken(secret string, duration time.Duration) Token {
	return &jwtToken{secret, duration}
}

func (t *jwtToken) Create(user *User) (string, error) {
	tokenID, err := uuid.NewRandom()
	if err != nil {
		return "", fmt.Errorf("create JWT ID: %w", err)
	}

	claims := userClaims{
		UserID: user.ID,
		RegisteredClaims: jwtlib.RegisteredClaims{
			ID:        tokenID.String(),
			Subject:   strconv.FormatUint(user.ID, 10),
			IssuedAt:  jwtlib.NewNumericDate(time.Now()),
			ExpiresAt: jwtlib.NewNumericDate(time.Now().Add(t.duration)),
		},
	}

	token := jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, &claims)
	return token.SignedString([]byte(t.secret))
}

func (t *jwtToken) Verify(token string, user *User) error {
	tkn, err := jwtlib.ParseWithClaims(token, &userClaims{}, func(token *jwtlib.Token) (any, error) {
		if _, ok := token.Method.(*jwtlib.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("invalid token signing method")
		}
		return []byte(t.secret), nil
	})
	if err != nil {
		return fmt.Errorf("parse with claims: %w", err)
	}

	claims, ok := tkn.Claims.(*userClaims)
	if !ok {
		return fmt.Errorf("invalid token claims")
	}
	user.ID = claims.UserID
	return nil
}
