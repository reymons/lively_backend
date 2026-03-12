package handler

import (
	"net/http"

	"lively/core/model"
	"lively/db"
	"lively/security/jwt"
	"lively/store"
	"lively/transport/http/codec"
)

type getMeRes struct {
	ID       uint64 `json:"id"`
	Username string `json:"username"`
}

type Users struct {
	users    store.Users
	dbClient db.Client
}

func NewUsers(users store.Users, dbClient db.Client) *Users {
	return &Users{users, dbClient}
}

func (h *Users) GetCurrent(w http.ResponseWriter, req *http.Request) {
	jwtUser, ok := jwt.GetUser(w, req)
	if !ok {
		return
	}
	var user model.User
	if err := h.users.GetByID(req.Context(), h.dbClient, jwtUser.ID, &user); err != nil {
		sendHttpError(w, req, "get user by ID", err)
		return
	}
	codec.EncodeBody(w, http.StatusOK, &getMeRes{
		ID:       user.ID,
		Username: user.Username,
	})
}
