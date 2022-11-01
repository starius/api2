package api2

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/starius/api2"
	"go.uber.org/goleak"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

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
		{Method: http.MethodGet, Path: "/number", Handler: getHandler},
		{Method: http.MethodPost, Path: "/number", Handler: postHandler},
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

func TestQueryAndHeader(t *testing.T) {
	type PostRequest struct {
		JsonField    int `json:"json_field"`
		QueryField   int `query:"query_field"`
		HeaderField  int `header:"query_field"`
		CookieField  int `cookie:"cookie_field"`
		SkippedField int `json:"-"`
	}
	type PostResponse struct {
		JsonField    int         `json:"json_field"`
		HeaderField  int         `header:"query_field"`
		CookieField  http.Cookie `cookie:"cookie_field"`
		SkippedField int         `json:"-"`
	}

	postHandler := func(ctx context.Context, req *PostRequest) (res *PostResponse, err error) {
		if req.SkippedField != 0 {
			return nil, fmt.Errorf("SkippedField=%d, want 0", req.SkippedField)
		}
		return &PostResponse{
			JsonField:   req.JsonField,
			HeaderField: req.QueryField + req.HeaderField + req.CookieField,
			CookieField: http.Cookie{
				Name:  "cookie_field",
				Value: "my cookie",
				Path:  "/path",
			},
			SkippedField: 5,
		}, nil
	}

	routes := []api2.Route{
		{Method: http.MethodPost, Path: "/number", Handler: postHandler, Transport: &api2.JsonTransport{}},
	}

	mux := http.NewServeMux()
	api2.BindRoutes(mux, routes)
	server := httptest.NewServer(mux)
	defer server.Close()

	client := api2.NewClient(routes, server.URL)

	ctx := context.Background()

	postRes := &PostResponse{}
	err := client.Call(ctx, postRes, &PostRequest{
		JsonField:    1,
		QueryField:   2,
		HeaderField:  3,
		CookieField:  4,
		SkippedField: 5,
	})
	if err != nil {
		t.Errorf("POST failed: %v.", err)
	}

	if postRes.JsonField != 1 {
		t.Errorf("JsonField=%d, want 1", postRes.JsonField)
	}
	if postRes.HeaderField != 2+3+4 {
		t.Errorf("HeaderField=%d, want %d", postRes.HeaderField, 2+3+4)
	}
	if postRes.CookieField.Value != "my cookie" {
		t.Errorf("CookieField.Value=%q, want %q", postRes.CookieField.Value, "my cookie")
	}
	if postRes.SkippedField != 0 {
		t.Errorf("SkippedField=%d, want 0", postRes.SkippedField)
	}
}

type customTransport struct {
	used bool
}

func (t *customTransport) RoundTrip(req *http.Request) (res *http.Response, err error) {
	t.used = true
	return http.DefaultTransport.RoundTrip(req)
}

func TestCustomClient(t *testing.T) {
	type GetRequest struct {
	}
	type GetResponse struct {
	}

	getHandler := func(ctx context.Context, req *GetRequest) (res *GetResponse, err error) {
		return &GetResponse{}, nil
	}

	routes := []api2.Route{
		{Method: http.MethodGet, Path: "/number", Handler: getHandler},
	}

	mux := http.NewServeMux()
	api2.BindRoutes(mux, routes)
	server := httptest.NewServer(mux)
	defer server.Close()

	transport := &customTransport{}
	httpClient := &http.Client{
		Transport: transport,
	}

	client := api2.NewClient(routes, server.URL, api2.CustomClient(httpClient))

	getRes := &GetResponse{}
	err := client.Call(context.Background(), getRes, &GetRequest{})
	if err != nil {
		t.Errorf("GET(10) failed: %v.", err)
	}
	if !transport.used {
		t.Errorf("Custom transport has not been used")
	}
}
