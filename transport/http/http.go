package http

import (
	"net/http"
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

func (t *Transport) RunServer(addr string) error {
	mux := http.NewServeMux()

	for _, h := range t.handlers {
		mux.Handle(h.Path, h.Handler)
	}

	srv := http.Server{
		Addr:    addr,
		Handler: mux,
	}

	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		return err
	}
	return nil
}
