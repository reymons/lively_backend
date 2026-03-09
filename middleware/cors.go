package middleware

import (
	"net/http"
	"slices"
	"strconv"
	"strings"
)

type CORSConfig struct {
	Origins     []string
	Methods     []string
	Headers     []string
	Credentials bool
	MaxAge      int64
}

var defaultMethods = []string{
	http.MethodGet,
	http.MethodPost,
	http.MethodHead,
}

func CORS(next http.Handler, cfg CORSConfig) http.Handler {
	if len(cfg.Methods) == 0 {
		cfg.Methods = slices.Clone(defaultMethods)
	}

	allowedMethods := strings.Join(cfg.Methods, ", ")
	allowedHeaders := strings.Join(cfg.Headers, ", ")

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		org := req.Header.Get("Origin")

		// Nothing we can do if Origin is empty.
		if org == "" {
			next.ServeHTTP(w, req)
			return
		}

		if cfg.Origins[0] != "*" && !slices.Contains(cfg.Origins, org) {
			w.WriteHeader(http.StatusOK)
			return
		}

		method := req.Method
		corsPreflight := false

		if req.Method == http.MethodOptions {
			// CORS preflights always use this header
			m := req.Header.Get("Access-Control-Request-Method")

			// This means it's just a random OPTIONS request,
			// so we just feed it to our router
			if m == "" {
				next.ServeHTTP(w, req)
				return
			}

			method = m
			corsPreflight = true
		}

		if len(cfg.Origins) != 1 || len(cfg.Origins) == 1 && cfg.Origins[0] != "*" {
			if !slices.Contains(cfg.Methods, method) {
				w.WriteHeader(http.StatusOK)
				return
			}
		}

		// Everything's fine, now it's time to set
		// all the Access-Control-* headers

		if corsPreflight {
			if cfg.MaxAge > 0 {
				w.Header().Set("Access-Control-Max-Age", strconv.FormatInt(cfg.MaxAge, 10))
			}
			if allowedHeaders != "" {
				w.Header().Set("Access-Control-Allow-Headers", allowedHeaders)
			}
			w.Header().Set("Access-Control-Allow-Methods", allowedMethods)
		}

		if cfg.Credentials {
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}

		w.Header().Set("Access-Control-Allow-Origin", org)

		if corsPreflight {
			w.WriteHeader(http.StatusOK)
		} else {
			next.ServeHTTP(w, req)
		}
	})
}
