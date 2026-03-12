package jwt

import (
	"context"
	"net/http"
	"strings"
)

const ctxKey = "jwt-user"

type AuthMiddleware struct {
	jwtService Service
}

func NewAuthMiddleware(jwtService Service) *AuthMiddleware {
	return &AuthMiddleware{jwtService}
}

func (m *AuthMiddleware) getBearerToken(r *http.Request) string {
	prefix := "Bearer "
	hdr := r.Header.Get("Authorization")
	if !strings.HasPrefix(hdr, prefix) {
		return ""
	}
	return strings.TrimPrefix(hdr, prefix)
}

type httpHandler = func(http.ResponseWriter, *http.Request)

func (m *AuthMiddleware) Wrap(handler httpHandler) httpHandler {
	return func(w http.ResponseWriter, req *http.Request) {
		token := m.getBearerToken(req)
		if token == "" {
			http.Error(w, "", http.StatusUnauthorized)
			return
		}
		var user User
		if err := m.jwtService.AccessToken().Verify(token, &user); err != nil {
			http.Error(w, "", http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(req.Context(), ctxKey, user)
		handler(w, req.WithContext(ctx))
	}
}

func GetUser(w http.ResponseWriter, req *http.Request) (User, bool) {
	user, ok := req.Context().Value(ctxKey).(User)
	if !ok {
		http.Error(w, "jwt user missing", http.StatusUnauthorized)
	}
	return user, ok
}
