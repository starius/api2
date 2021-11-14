package api2

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/starius/api2"
)

func TestMaxBody(t *testing.T) {
	type Request struct {
		Foo string `json:"foo"`
	}
	type Response struct {
		Bar string `json:"bar"`
	}

	const size = 1000

	handler := func(ctx context.Context, req *Request) (res *Response, err error) {
		return &Response{
			Bar: string(make([]byte, size)),
		}, nil
	}

	routes := []api2.Route{
		{Method: http.MethodPost, Path: "/handle", Handler: handler},
	}

	t.Run("no maxBody (control)", func(t *testing.T) {
		mux := http.NewServeMux()
		api2.BindRoutes(mux, routes)
		server := httptest.NewServer(mux)
		defer server.Close()

		client := api2.NewClient(routes, server.URL)

		res := &Response{}
		err := client.Call(context.Background(), res, &Request{
			Foo: string(make([]byte, size)),
		})
		if err != nil {
			t.Errorf("request failed: %v.", err)
		}
		if len(res.Bar) != size {
			t.Errorf("wrong bar size: got %d, want %d", len(res.Bar), size)
		}
	})

	t.Run("maxBody in server", func(t *testing.T) {
		mux := http.NewServeMux()
		api2.BindRoutes(mux, routes, api2.MaxBody(size/2))
		server := httptest.NewServer(mux)
		defer server.Close()

		client := api2.NewClient(routes, server.URL)

		res := &Response{}
		err := client.Call(context.Background(), res, &Request{
			Foo: string(make([]byte, size)),
		})
		if err == nil {
			t.Errorf("request had to fail, but passed")
		}
		wantMessage := "API returned error with HTTP status 400 Bad Request: failed to parse request: http: request body too large"
		if err.Error() != wantMessage {
			t.Errorf("request failed with unexpected error: got %q, want %q", err.Error(), wantMessage)
		}
	})

	t.Run("maxBody in client", func(t *testing.T) {
		mux := http.NewServeMux()
		api2.BindRoutes(mux, routes)
		server := httptest.NewServer(mux)
		defer server.Close()

		client := api2.NewClient(routes, server.URL, api2.MaxBody(size/2))

		res := &Response{}
		err := client.Call(context.Background(), res, &Request{
			Foo: string(make([]byte, size)),
		})
		if err == nil {
			t.Errorf("request had to fail, but passed")
		}
		wantMessage := "http: request body too large"
		if err.Error() != wantMessage {
			t.Errorf("request failed with unexpected error: got %q, want %q", err.Error(), wantMessage)
		}
	})
}
