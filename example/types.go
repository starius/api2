package example

import (
	"context"
	"net/http"

	"github.com/starius/api2"
)

type EchoRequest struct {
	Text string `json:"text"`
}

type EchoResponse struct {
	Text string `json:"text"`
}

type EchoService interface {
	Echo(ctx context.Context, req *EchoRequest) (*EchoResponse, error)
}

func GetRoutes(s EchoService) []api2.Route {
	return []api2.Route{
		{http.MethodPost, "/echo", api2.Method(&s, "Echo"), &api2.JsonTransport{}},
	}
}
