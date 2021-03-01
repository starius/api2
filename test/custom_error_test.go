package api2

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/starius/api2"
)

type MyError struct {
	MyCode int
}

func (e MyError) Error() string {
	return "my error"
}

func TestErrorType(t *testing.T) {
	type HelloRequest struct {
		Ok                  bool
		MyError             bool
		WrappedError        bool
		DoublyWrappedError  bool
		GenericError        bool
		WrappedGenericError bool
	}
	type HelloResponse struct {
	}

	helloHandler := func(ctx context.Context, req *HelloRequest) (res *HelloResponse, err error) {
		if req.Ok {
			return &HelloResponse{}, nil
		}
		if req.MyError {
			return nil, MyError{MyCode: 123}
		}
		if req.WrappedError {
			return nil, fmt.Errorf("failed: %w", MyError{MyCode: 123})
		}
		if req.DoublyWrappedError {
			return nil, fmt.Errorf("failed: %w", fmt.Errorf("error: %w", MyError{MyCode: 123}))
		}
		if req.GenericError {
			return nil, errors.New("generic error")
		}
		if req.WrappedGenericError {
			return nil, fmt.Errorf("error: %w", errors.New("generic error"))
		}
		panic("bad input")
	}

	routes := []api2.Route{
		{
			Method:  http.MethodPost,
			Path:    "/hello",
			Handler: helloHandler,
			Transport: &api2.JsonTransport{
				Errors: map[string]error{
					"MyError": MyError{},
				},
			},
		},
	}

	mux := http.NewServeMux()
	api2.BindRoutes(mux, routes)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	client := api2.NewClient(routes, server.URL)

	ctx := context.Background()

	t.Run("ok", func(t *testing.T) {
		helloRes := &HelloResponse{}
		err := client.Call(ctx, helloRes, &HelloRequest{Ok: true})
		if err != nil {
			t.Errorf("Hello failed: %v.", err)
		}
	})

	t.Run("my error", func(t *testing.T) {
		requests := map[string]*HelloRequest{
			"my error":             &HelloRequest{MyError: true},
			"wrapped error":        &HelloRequest{WrappedError: true},
			"doubly wrapped error": &HelloRequest{DoublyWrappedError: true},
		}
		for name, req := range requests {
			t.Run(name, func(t *testing.T) {
				helloRes := &HelloResponse{}
				err := client.Call(ctx, helloRes, req)
				if err == nil {
					t.Fatalf("Hello did not fail.")
				}
				myErr, ok := err.(MyError)
				if !ok {
					t.Fatalf("The error is not MyError.")
				}
				if myErr.MyCode != 123 {
					t.Fatalf("MyCode is %d, want %d.", myErr.MyCode, 123)
				}
			})
		}
	})

	t.Run("generic error", func(t *testing.T) {
		requests := map[string]*HelloRequest{
			"generic error":         &HelloRequest{GenericError: true},
			"wrapped generic error": &HelloRequest{WrappedGenericError: true},
		}
		for name, req := range requests {
			t.Run(name, func(t *testing.T) {
				helloRes := &HelloResponse{}
				err := client.Call(ctx, helloRes, req)
				if err == nil {
					t.Fatalf("Hello did not fail.")
				}
				if !strings.Contains(err.Error(), "generic error") {
					t.Fatalf("Error is %q, should contain %q.", err.Error(), "generic error")
				}
				if _, ok := err.(MyError); ok {
					t.Fatalf("The error is MyError.")
				}
			})
		}
	})
}
