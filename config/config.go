package config

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"strconv"
	"strings"
)

func fatal(prfx string, envVar string, val any) {
	log.Fatal(fmt.Sprintf("ERROR: %s is invalid, %s = %+v", prfx, envVar, val))
}

func isValidPort(port string) bool {
	val, err := strconv.Atoi(port)
	if err != nil {
		return false
	}
	return val >= 1 && val <= 65535
}

type Config struct {
	DatabaseURL          string
	HTTPServerHost       string
	HTTPServerPort       string
	HTTPAllowedOrigins   []string
	RTMPServerHost       string
	RTMPServerPort       string
	JWTAccessTokenSecret string
}

func NewConfig() *Config {
	conf := &Config{}

	host := os.Getenv("HTTP_SERVER_HOST")
	if host == "" {
		fatal("HTTP server host", "HTTP_SERVER_HOST", host)
	}
	conf.HTTPServerHost = host

	port := os.Getenv("HTTP_SERVER_PORT")
	if !isValidPort(port) {
		fatal("HTTP server port", "HTTP_SERVER_PORT", port)
	}
	conf.HTTPServerPort = port

	host = os.Getenv("RTMP_SERVER_HOST")
	if host == "" {
		fatal("RTMP server host", "RTMP_SERVER_HOST", host)
	}
	conf.RTMPServerHost = host

	port = os.Getenv("RTMP_SERVER_PORT")
	if !isValidPort(port) {
		fatal("RTMP server port", "RTMP_SERVER_PORT", port)
	}
	conf.RTMPServerPort = port

	dbUser := os.Getenv("DB_USER")
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbName := os.Getenv("DB_NAME")
	dbPswd := url.QueryEscape(os.Getenv("DB_PASSWORD"))
	dbSSLMode := os.Getenv("DB_SSL_MODE")
	if dbUser == "" {
		fatal("Database user", "DB_USER", host)
	}
	if dbPswd == "" {
		fatal("Database password", "DB_PASSWORD", host)
	}
	if dbHost == "" {
		fatal("Database host", "DB_HOST", host)
	}
	if dbName == "" {
		fatal("Database name", "DB_NAME", host)
	}
	if !isValidPort(dbPort) {
		fatal("Database port", "DB_PORt", host)
	}
	if dbSSLMode != "disable" && dbSSLMode != "enable" {
		fatal("Database SSL mode", "DB_SSL_MODE", dbSSLMode)
	}
	conf.DatabaseURL = fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		dbUser, dbPswd, dbHost, dbPort, dbName, dbSSLMode,
	)

	conf.HTTPAllowedOrigins = strings.Split(os.Getenv("HTTP_ALLOWED_ORIGINS"), ",")

	secret := os.Getenv("JWT_ACCESS_TOKEN_SECRET")
	if secret == "" {
		fatal("JWT access token secret", "JWT_ACCESS_TOKEN_SECRET", secret)
	}
	conf.JWTAccessTokenSecret = secret

	return conf
}
