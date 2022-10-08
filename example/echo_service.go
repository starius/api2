package example

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"time"

	"google.golang.org/protobuf/types/known/durationpb"
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

func (s *EchoService) StreamBody(ctx context.Context, req *StreamRequest) (*StreamResponse, error) {
	var b bytes.Buffer
	for {
		buf := make([]byte, 1024)
		n, err := req.Body.Read(buf)
		b.Write(buf[:n])
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
	}
	defer req.Body.Close()
	return &StreamResponse{}, nil
}

func (s *EchoService) Since(ctx context.Context, req *SinceRequest) (*SinceResponse, error) {
	if !s.sessions[req.Session] {
		return nil, fmt.Errorf("bad session")
	}
	t1 := time.Date(2010, time.July, 10, 11, 30, 0, 0, time.UTC)
	t2 := req.Body.AsTime()
	duration := t2.Sub(t1)
	return &SinceResponse{
		Body: durationpb.New(duration),
	}, nil
}
