package api2

import (
	"log"
	"net/http"
)

type HttpClient interface {
	Do(req *http.Request) (*http.Response, error)
	CloseIdleConnections()
}

type Config struct {
	errorf        func(format string, args ...interface{})
	authorization string // Affects only clients.
	client        HttpClient
	maxBody       int64
}

const defaultMaxBody = 10 * 1024 * 1024

func NewDefaultConfig() *Config {
	return &Config{
		errorf:  log.Printf,
		maxBody: defaultMaxBody,
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

func CustomClient(client HttpClient) Option {
	return func(config *Config) {
		config.client = client
	}
}

func MaxBody(maxBody int64) Option {
	return func(config *Config) {
		config.maxBody = maxBody
	}
}
