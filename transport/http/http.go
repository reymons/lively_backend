package http

import (
	"net/http"

	"lively/middleware"
)

type HandlerInfo struct {
	Path    string
	Handler http.Handler
}

type NewTransportConfig struct {
	Host     string
	Port     string
	Handlers []HandlerInfo
}

type Transport struct {
	handlers []HandlerInfo
}

func NewTransport(handlers []HandlerInfo) *Transport {
	return &Transport{
		handlers: handlers,
	}
}

func (t *Transport) RunServer(addr string, allowedOrigins []string) error {
	mux := http.NewServeMux()

	for _, h := range t.handlers {
		mux.Handle(h.Path, h.Handler)
	}

	h := http.Handler(mux)
	h = middleware.Logger(h)
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

	srv := http.Server{
		Addr:    addr,
		Handler: h,
	}

	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		return err
	}
	return nil
}
