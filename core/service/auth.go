package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"lively/core"
	"lively/core/model"
	"lively/db"
	"lively/security/password"
	"lively/store"
)

type Auth interface {
	SignIn(ctx context.Context, username, password string, user *model.User) error

	SignUp(ctx context.Context, username, password string, user *model.User) error
}

type authService struct {
	dbClient    db.Client
	users       store.Users
	streamKeys  store.StreamKeys
	pswdManager password.Manager
}

func NewAuth(
	dbClient db.Client,
	users store.Users,
	streamKeys store.StreamKeys,
	pswdManager password.Manager,
) Auth {
	return &authService{dbClient, users, streamKeys, pswdManager}
}

func (s *authService) createStreamKey() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "sk_" + hex.EncodeToString(b), nil
}

func (s *authService) SignIn(ctx context.Context, username, password string, user *model.User) error {
	if err := s.users.GetByUsername(ctx, s.dbClient, username, user); err != nil {
		return fmt.Errorf("get client by email: %w", err)
	}
	if !s.pswdManager.VerifyPassword(password, user.Password) {
		return core.ErrInvalidCredentials
	}
	return nil
}

func (s *authService) SignUp(ctx context.Context, username, password string, user *model.User) error {
	if s.users.ExistsByUsername(ctx, s.dbClient, username) {
		return core.ErrUsernameTaken
	}

	hashed, err := s.pswdManager.HashPassword(password)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	key, err := s.createStreamKey()
	if err != nil {
		return fmt.Errorf("create stream key: %w", err)
	}

	return s.dbClient.ExecTrx(ctx, func(trx db.Trx) error {
		u := model.User{Username: username, Password: hashed}
		if err := s.users.Save(ctx, trx, &u); err != nil {
			return fmt.Errorf("save user: %w", err)
		}
		sk := model.StreamKey{
			Key:    key,
			UserID: u.ID,
			Active: true,
		}
		if err := s.streamKeys.Save(ctx, trx, &sk); err != nil {
			return fmt.Errorf("save stream key: %w", err)
		}
		*user = u
		return nil
	})
}
