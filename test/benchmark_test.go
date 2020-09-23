package api2

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/starius/api2"
)

func BenchmarkAPI(b *testing.B) {
	type PostRequest struct {
		Foo    int    `json:"foo"`
		Bar    string `json:"bar"`
		Baz    []int  `json:"baz"`
		PageID string `json:"page_id"`
	}
	type PostResponse struct {
		Counters   map[string]int `json:"counters"`
		NextPageID string         `json:"next_page_id"`
	}

	postHandler := func(ctx context.Context, req *PostRequest) (res *PostResponse, err error) {
		counters := make(map[string]int, len(req.Baz))
		for i, count := range req.Baz {
			counters[strconv.Itoa(i)] = count
		}
		return &PostResponse{
			Counters:   counters,
			NextPageID: req.PageID + "1",
		}, nil
	}

	routes := []api2.Route{
		{Method: http.MethodPost, Path: "/number", Handler: postHandler},
	}

	mux := http.NewServeMux()
	api2.BindRoutes(mux, routes)
	server := httptest.NewServer(mux)
	defer server.Close()

	client := api2.NewClient(routes, server.URL)

	ctx := context.Background()

	for i := 0; i < b.N; i++ {
		postReq := &PostRequest{
			Foo:    123,
			Bar:    "bar bar bar",
			Baz:    []int{10, 30, 25, 0},
			PageID: "page-4cdab3",
		}
		postRes := &PostResponse{}
		err := client.Call(ctx, postRes, postReq)
		if err != nil {
			b.Errorf("POST failed: %v.", err)
		}
	}
}
