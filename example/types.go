package example

//go:generate go run ./gen/...

import (
	"context"
	"net/http"

	"github.com/starius/api2"
)

type HelloRequest struct {
	Key string `query:"key"`
}

type HelloResponse struct {
	Session string `header:"session"`
}

type EchoRequest struct {
	Session string `header:"session"`
	Text    string `json:"text"`
}

type EchoResponse struct {
	Text string `json:"text"`
}

type EchoService interface {
	Hello(ctx context.Context, req *HelloRequest) (*HelloResponse, error)
	Echo(ctx context.Context, req *EchoRequest) (*EchoResponse, error)
}

func GetRoutes(s EchoService) []api2.Route {
	return []api2.Route{
		{Method: http.MethodPost, Path: "/hello", Handler: api2.Method(&s, "Hello")},
		{Method: http.MethodPost, Path: "/echo", Handler: api2.Method(&s, "Echo")},
	}
}
