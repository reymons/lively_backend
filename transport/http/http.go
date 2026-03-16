package http

import (
	"net/http"

	"lively/core/service"
	"lively/db"
	"lively/middleware"
	"lively/security/jwt"
	"lively/store"
	"lively/transport/http/handler"
)

type Dependencies struct {
	// Stores
	Users store.Users
	// Core services
	AuthService      service.Auth
	StreamKeyService service.StreamKey
	// Misc
	JWTService jwt.Service
	DBClient   db.Client
}

type HandlerInfo struct {
	Path    string
	Handler http.Handler
}

type Transport struct {
	handlers []HandlerInfo
	deps     Dependencies
}

func NewTransport(handlers []HandlerInfo, deps *Dependencies) *Transport {
	return &Transport{
		handlers: handlers,
		deps:     *deps,
	}
}

func (t *Transport) RunServer(addr string, allowedOrigins []string) error {
	// Misc
	authMiddleware := jwt.NewAuthMiddleware(t.deps.JWTService)

	// Handlers
	authHandler := handler.NewAuth(t.deps.AuthService, t.deps.JWTService)
	usersHandler := handler.NewUsers(t.deps.Users, t.deps.DBClient)
	skHandler := handler.NewStreamKeys(t.deps.StreamKeyService)

	// Multiplexer set-up
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/auth/sign-up", authHandler.SignUp)
	mux.HandleFunc("POST /api/v1/auth/sign-in", authHandler.SignIn)
	mux.HandleFunc("GET /api/v1/users/current", authMiddleware.Wrap(usersHandler.GetCurrent))
	mux.HandleFunc("GET /api/v1/users/usernames/", usersHandler.GetByUsername)
	mux.HandleFunc("GET /api/v1/stream-keys/current", authMiddleware.Wrap(skHandler.GetOfCurrentUser))

	for _, h := range t.handlers {
		mux.Handle(h.Path, h.Handler)
	}

	h := http.Handler(mux)
	h = middleware.CORS(h, middleware.CORSConfig{
		Credentials: true,
		Origins:     allowedOrigins,
		MaxAge:      300, // 5 min
		Headers:     []string{"Content-Type", "Authorization"},
		Methods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodHead,
			http.MethodPatch,
			http.MethodPut,
			http.MethodDelete,
		},
	})
	h = middleware.Logger(h)
	h = middleware.RealIP(h)

	srv := http.Server{
		Addr:    addr,
		Handler: h,
	}

	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		return err
	}
	return nil
}
