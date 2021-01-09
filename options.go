package api2

import "log"

type Config struct {
	errorf        func(format string, args ...interface{})
	authorization string // Affects only clients.
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
