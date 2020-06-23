package main

import (
	"context"
	"log"
	"net/http"

	"github.com/starius/api2"
	"github.com/starius/api2/example"
)

type EchoService struct {
}

func (s *EchoService) Echo(ctx context.Context, req *example.EchoRequest) (*example.EchoResponse, error) {
	return &example.EchoResponse{
		Text: req.Text,
	}, nil
}

func main() {
	service := &EchoService{}
	routes := example.GetRoutes(service)
	api2.BindRoutes(http.DefaultServeMux, routes)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
