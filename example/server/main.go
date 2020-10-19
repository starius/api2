package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"

	"github.com/starius/api2"
	"github.com/starius/api2/example"
)

type EchoService struct {
	sessions map[string]bool
}

func NewEchoService() *EchoService {
	return &EchoService{
		sessions: make(map[string]bool),
	}
}

func (s *EchoService) Hello(ctx context.Context, req *example.HelloRequest) (*example.HelloResponse, error) {
	if req.Key != "secret password" {
		return nil, fmt.Errorf("bad key")
	}
	sessionBytes := make([]byte, 16)
	if _, err := rand.Read(sessionBytes); err != nil {
		return nil, err
	}
	session := hex.EncodeToString(sessionBytes)
	s.sessions[session] = true
	return &example.HelloResponse{
		Session: session,
	}, nil
}

func (s *EchoService) Echo(ctx context.Context, req *example.EchoRequest) (*example.EchoResponse, error) {
	if !s.sessions[req.Session] {
		return nil, fmt.Errorf("bad session")
	}
	return &example.EchoResponse{
		Text: req.Text,
	}, nil
}

func main() {
	service := NewEchoService()
	routes := example.GetRoutes(service)
	api2.BindRoutes(http.DefaultServeMux, routes)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
