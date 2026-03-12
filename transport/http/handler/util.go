package handler

import (
	"errors"
	"log"
	"net/http"

	"lively/core"
)

func logError(err error, prefix string, req *http.Request) {
	log.Printf(
		"ERROR: %s - %s %s %s, status: %d, %s: %s\n",
		req.RemoteAddr,
		req.Method,
		req.RequestURI,
		req.Proto,
		http.StatusInternalServerError,
		prefix,
		err.Error(),
	)
}

// Maps core errors to HTTP ones and sends them to the client
// If an error can't be mapped, it sends 500 status code by default and logs the error
// A prefix can be passed to form a better error stack trace
func sendHttpError(
	w http.ResponseWriter,
	req *http.Request,
	prefix string,
	err error,
) {
	switch {
	case errors.Is(err, core.ErrEntityNotFound):
		http.Error(w, "", http.StatusNotFound)
	case errors.Is(err, core.ErrInvalidCredentials):
		http.Error(w, "Invalid credentials", http.StatusBadRequest)
	case errors.Is(err, core.ErrUsernameTaken):
		http.Error(w, "Username is taken", http.StatusBadRequest)
	default:
		http.Error(w, "", http.StatusInternalServerError)
		logError(err, prefix, req)
	}
}
