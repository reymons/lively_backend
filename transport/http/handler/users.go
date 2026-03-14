package handler

import (
	"net/http"
	"strings"

	"lively/core/model"
	"lively/db"
	"lively/security/jwt"
	"lively/store"
	"lively/transport/http/codec"
)

type userRes struct {
	ID       uint64 `json:"id"`
	Username string `json:"username"`
}

func (res *userRes) FromUser(u *model.User) {
	res.ID = u.ID
	res.Username = u.Username
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
	var res userRes
	res.FromUser(&user)
	codec.EncodeBody(w, http.StatusOK, &res)
}

func (h *Users) GetByUsername(w http.ResponseWriter, req *http.Request) {
	username := strings.TrimPrefix(req.URL.Path, "/api/v1/users/usernames/")
	var user model.User
	if err := h.users.GetByUsername(req.Context(), h.dbClient, username, &user); err != nil {
		sendHttpError(w, req, "get user by ID", err)
		return
	}
	var res userRes
	res.FromUser(&user)
	codec.EncodeBody(w, http.StatusOK, &res)
}
