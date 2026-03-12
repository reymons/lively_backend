package store

import (
	"context"

	"lively/core/model"
	"lively/db"
)

type Users interface {
	GetByID(ctx context.Context, dbClient db.DB, id uint64, user *model.User) error

	GetByUsername(ctx context.Context, dbClient db.DB, username string, user *model.User) error

	Save(ctx context.Context, dbClient db.DB, user *model.User) error

	ExistsByUsername(ctx context.Context, dbClient db.DB, username string) bool
}

type usersStore struct{}

func NewUsers() Users {
	return &usersStore{}
}

func (s *usersStore) GetByID(ctx context.Context, dbClient db.DB, id uint64, u *model.User) error {
	row := dbClient.QueryRowContext(ctx, "SELECT id, username, password, created_at FROM users WHERE id = $1", id)
	err := row.Scan(&u.ID, &u.Username, &u.Password, &u.CreatedAt)
	return db.MapError(err)
}

func (s *usersStore) GetByUsername(ctx context.Context, dbClient db.DB, username string, u *model.User) error {
	row := dbClient.QueryRowContext(ctx, "SELECT id, username, password, created_at FROM users WHERE username = $1", username)
	err := row.Scan(&u.ID, &u.Username, &u.Password, &u.CreatedAt)
	return db.MapError(err)
}

func (s *usersStore) Save(ctx context.Context, dbClient db.DB, u *model.User) error {
	row := dbClient.QueryRowContext(
		ctx, "INSERT INTO users(username, password) VALUES ($1,$2) RETURNING id, created_at",
		u.Username, u.Password,
	)
	return db.MapError(row.Scan(&u.ID, &u.CreatedAt))
}

func (s *usersStore) ExistsByUsername(ctx context.Context, dbClient db.DB, username string) bool {
	row := dbClient.QueryRowContext(ctx, "SELECT count(1) FROM users WHERE username = $1", username)
	var count int
	if err := row.Scan(&count); err != nil {
		return false
	}
	return count > 0
}
