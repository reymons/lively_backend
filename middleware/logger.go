package middleware

import (
	"log"
	"net/http"
)

func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		log.Printf(
			"INFO: HTTP: %s - %s %s %s\n",
			req.RemoteAddr,
			req.Method,
			req.RequestURI,
			req.Proto,
		)

		next.ServeHTTP(w, req)
	})
}
