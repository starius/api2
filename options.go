package api2

import (
	"log"
	"net/http"
)

type Config struct {
	errorf        func(format string, args ...interface{})
	authorization string // Affects only clients.
	client        *http.Client
}

func NewDefaultConfig() *Config {
	return &Config{
		errorf: log.Printf,
	}
}

type Option func(*Config)

func ErrorLogger(logger func(format string, args ...interface{})) Option {
	return func(config *Config) {
		config.errorf = logger
	}
}

func AuthorizationHeader(authorization string) Option {
	return func(config *Config) {
		config.authorization = authorization
	}
}

func CustomClient(client *http.Client) Option {
	return func(config *Config) {
		config.client = client
	}
}
