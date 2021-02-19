package api2

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/starius/api2"
)

func TestCtx(t *testing.T) {
	type HelloRequest struct {
	}
	type HelloResponse struct {
	}

	var cancelled int64
	var wg sync.WaitGroup

	helloHandler := func(ctx context.Context, req *HelloRequest) (res *HelloResponse, err error) {
		defer wg.Done()
		timer := time.NewTimer(2 * time.Second)
		select {
		case <-ctx.Done():
			atomic.StoreInt64(&cancelled, 1)
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
			return &HelloResponse{}, nil
		}
	}

	routes := []api2.Route{
		{
			Method:  http.MethodPost,
			Path:    "/hello",
			Handler: helloHandler,
		},
	}

	mux := http.NewServeMux()
	api2.BindRoutes(mux, routes)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	client := api2.NewClient(routes, server.URL)

	t.Run("no cancel", func(t *testing.T) {
		ctx := context.Background()

		atomic.StoreInt64(&cancelled, 0)

		helloRes := &HelloResponse{}
		wg.Add(1)
		err := client.Call(ctx, helloRes, &HelloRequest{})
		wg.Wait()
		if err != nil {
			t.Errorf("Hello failed: %v.", err)
		}
		if atomic.LoadInt64(&cancelled) == 1 {
			t.Errorf("the request was unexpectedly cancelled")
		}
	})

	t.Run("cancellation", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second/10)
		defer cancel()

		atomic.StoreInt64(&cancelled, 0)

		helloRes := &HelloResponse{}
		wg.Add(1)
		err := client.Call(ctx, helloRes, &HelloRequest{})
		wg.Wait()
		if err == nil {
			t.Errorf("Hello did not failed.")
		}
		if atomic.LoadInt64(&cancelled) == 0 {
			t.Errorf("the request was not cancelled")
		}
	})
}
