package example

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
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
	if req.User != "good-user" {
		return nil, fmt.Errorf("bad user")
	}
	return &EchoResponse{
		Text: req.Text,
	}, nil
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

func (s *EchoService) Stream(ctx context.Context, req *StreamRequest) (*StreamResponse, error) {
	if !s.sessions[req.Session] {
		return nil, fmt.Errorf("bad session")
	}
	input, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read input: %w", err)
	}

	output := bytes.ToUpper(input)

	return &StreamResponse{
		Body: io.NopCloser(bytes.NewReader(output)),
	}, nil
}

func (s *EchoService) Redirect(ctx context.Context, req *RedirectRequest) (*RedirectResponse, error) {
	return &RedirectResponse{
		Status: http.StatusFound,
		URL:    fmt.Sprintf("https://example.com/user?id=%s", req.ID),
	}, nil
}

func (s *EchoService) Raw(ctx context.Context, req *RawRequest) (*RawResponse, error) {
	return &RawResponse{
		Token: req.Token,
	}, nil
}
