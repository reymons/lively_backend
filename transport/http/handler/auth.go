package handler

import (
	"net/http"
	"regexp"

	"lively/core/model"
	"lively/core/service"
	"lively/security/jwt"
	"lively/transport/http/codec"
)

type signUpReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

var usernameRx = regexp.MustCompile(`^[0-9a-zA-Z_-]+$`)

func (req *signUpReq) Valid() codec.Errors {
	errors := codec.Errors{}
	if req.Username == "" {
		errors["username"] = codec.ERequiredField
	} else if len(req.Username) < 4 || len(req.Username) > 40 {
		errors["username"] = "must be between 6 and 40 characters"
	} else if !usernameRx.MatchString(req.Username) {
		errors["username"] = "must contain only valid characters: a-z, A-Z, _, -, 0-9"
	}
	if req.Password == "" {
		errors["password"] = codec.ERequiredField
	} else if len(req.Password) < 6 || len(req.Password) > 100 {
		errors["password"] = "must be between 6 and 100 characters"
	}
	return errors
}

type signInReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (req *signInReq) Valid() codec.Errors {
	errors := codec.Errors{}
	if req.Username == "" {
		errors["username"] = codec.ERequiredField
	}
	if req.Password == "" {
		errors["password"] = codec.ERequiredField
	}
	return errors
}

type authRes struct {
	AccessToken string `json:"access_token"`
}

type Auth struct {
	authService service.Auth
	jwtService  jwt.Service
}

func NewAuth(authService service.Auth, jwtService jwt.Service) *Auth {
	return &Auth{authService, jwtService}
}

func (h *Auth) sendAuthResponse(w http.ResponseWriter, req *http.Request, userID uint64) {
	jwtUser := jwt.User{ID: userID}
	token, err := h.jwtService.AccessToken().Create(&jwtUser)
	if err != nil {
		sendHttpError(w, req, "create access token", err)
		return
	}
	res := authRes{AccessToken: token}
	codec.EncodeBody(w, http.StatusCreated, &res)
}

func (h *Auth) SignUp(w http.ResponseWriter, req *http.Request) {
	body, ok := codec.DecodeBody[*signUpReq](w, req)
	if !ok {
		return
	}
	var user model.User
	if err := h.authService.SignUp(req.Context(), body.Username, body.Password, &user); err != nil {
		sendHttpError(w, req, "sign up", err)
		return
	}
	h.sendAuthResponse(w, req, user.ID)
}

func (h *Auth) SignIn(w http.ResponseWriter, req *http.Request) {
	body, ok := codec.DecodeBody[*signInReq](w, req)
	if !ok {
		return
	}
	var user model.User
	if err := h.authService.SignIn(req.Context(), body.Username, body.Password, &user); err != nil {
		sendHttpError(w, req, "sign in", err)
		return
	}
	h.sendAuthResponse(w, req, user.ID)
}
