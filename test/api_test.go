package api2

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/starius/api2"
)

func TestAPI(t *testing.T) {
	badNumber := 42 // State of the server.

	type GetRequest struct {
		Number int `json:"number"`
	}
	type GetResponse struct {
		DoubleNumber int `json:"double_number"`
	}

	getHandler := func(ctx context.Context, req *GetRequest) (res *GetResponse, err error) {
		if req.Number == badNumber {
			return nil, fmt.Errorf("bad number")
		}
		return &GetResponse{
			DoubleNumber: req.Number * 2,
		}, nil
	}

	type PostRequest struct {
		BadNumber int `json:"bad_number"`
	}
	type PostResponse struct {
	}

	postHandler := func(ctx context.Context, req *PostRequest) (res *PostResponse, err error) {
		badNumber = req.BadNumber
		return &PostResponse{}, nil
	}

	routes := []api2.Route{
		{http.MethodGet, "/number", getHandler, &api2.JsonTransport{}},
		{http.MethodPost, "/number", postHandler, &api2.JsonTransport{}},
	}

	mux := http.NewServeMux()
	api2.BindRoutes(mux, routes)
	server := httptest.NewServer(mux)
	defer server.Close()

	client := api2.NewClient(routes, server.URL)

	getRes := &GetResponse{}
	err := client.Call(context.Background(), getRes, &GetRequest{
		Number: 10,
	})
	if err != nil {
		t.Errorf("GET(10) failed: %v.", err)
	}
	if getRes.DoubleNumber != 20 {
		t.Errorf("GET(10) returned %d, want %d.", getRes.DoubleNumber, 20)
	}

	err = client.Call(context.Background(), getRes, &GetRequest{
		Number: 42,
	})
	if err == nil {
		t.Errorf("GET(42) didn't fail. Want failure.")
	} else if !strings.Contains(err.Error(), "bad number") {
		t.Errorf("GET(42) failed with error %v. Want error including 'bad number'.", err)
	}

	postRes := &PostResponse{}
	err = client.Call(context.Background(), postRes, &PostRequest{
		BadNumber: 100,
	})
	if err != nil {
		t.Errorf("POST(100) failed: %v.", err)
	}

	err = client.Call(context.Background(), getRes, &GetRequest{
		Number: 100,
	})
	if err == nil {
		t.Errorf("GET(100) didn't fail. Want failure.")
	} else if !strings.Contains(err.Error(), "bad number") {
		t.Errorf("GET(100) failed with error %v. Want error including 'bad number'.", err)
	}

	err = client.Call(context.Background(), getRes, &GetRequest{
		Number: 42,
	})
	if err != nil {
		t.Errorf("GET(42) failed: %v.", err)
	}
	if getRes.DoubleNumber != 84 {
		t.Errorf("GET(42) returned %d, want %d.", getRes.DoubleNumber, 84)
	}
}
