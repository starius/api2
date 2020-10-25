package example

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

type EchoRepository struct {
}

func NewEchoRepository() *EchoRepository {
	return &EchoRepository{}
}

type Result struct {
	Data string
}

func (s *EchoRepository) Generate() (*Result, error) {
	sessionBytes := make([]byte, 16)
	if _, err := rand.Read(sessionBytes); err != nil {
		return nil, err
	}
	session := hex.EncodeToString(sessionBytes)
	return &Result{Data: session}, nil
}

type EchoService struct {
	sessions map[string]bool
	repo     *EchoRepository
}

func NewEchoService(repo *EchoRepository) *EchoService {
	return &EchoService{
		sessions: make(map[string]bool),
		repo:     repo,
	}
}

func (s *EchoService) Hello(ctx context.Context, req *HelloRequest) (*HelloResponse, error) {
	if req.Key != "secret password" {
		return nil, fmt.Errorf("bad key")
	}
	session, err := s.repo.Generate()
	if err != nil {
		return nil, err
	}
	s.sessions[session.Data] = true
	return &HelloResponse{
		Session: session.Data,
	}, nil
}

func (s *EchoService) Echo(ctx context.Context, req *EchoRequest) (*EchoResponse, error) {
	if !s.sessions[req.Session] {
		return nil, fmt.Errorf("bad session")
	}
	return &EchoResponse{
		Text: req.Text,
	}, nil
}
