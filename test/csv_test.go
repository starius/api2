package api2

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
	"testing"

	"github.com/starius/api2"
	"github.com/starius/api2/errors"
)

func TestCSV(t *testing.T) {
	type Request struct {
	}

	getHandler := func(ctx context.Context, req *Request) (res *api2.CsvResponse, err error) {
		rows := make(chan []string, 2)
		rows <- []string{"Alice", "31"}
		rows <- []string{"Bob", "13"}
		close(rows)
		return &api2.CsvResponse{
			HttpCode: http.StatusOK,
			HttpHeaders: http.Header{
				"test-header": []string{"test value"},
			},
			CsvHeader: []string{"Name", "Age"},
			Rows:      rows,
		}, nil
	}

	routes := []api2.Route{
		{Method: http.MethodGet, Path: "/csv", Handler: getHandler, Transport: api2.CsvTransport},
	}

	mux := http.NewServeMux()
	api2.BindRoutes(mux, routes)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	client := api2.NewClient(routes, server.URL)

	req := &Request{}
	res := &api2.CsvResponse{
		Rows: make(chan []string),
	}

	var wg sync.WaitGroup

	var results [][]string
	wg.Add(1)
	go func() {
		defer wg.Done()
		for row := range res.Rows {
			results = append(results, row)
		}
	}()

	err := client.Call(context.Background(), res, req)
	if err != nil {
		t.Errorf("request failed: %v.", err)
	}

	wg.Wait()

	if res.HttpCode != http.StatusOK {
		t.Errorf("got different HTTP status")
	}

	if res.HttpHeaders.Get("test-header") != "test value" {
		t.Errorf("got different HTTP headers")
	}

	wantHeader := []string{"Name", "Age"}
	if !reflect.DeepEqual(res.CsvHeader, wantHeader) {
		t.Errorf("got different CSV headers")
	}

	wantRows := [][]string{
		{"Alice", "31"},
		{"Bob", "13"},
	}
	if !reflect.DeepEqual(results, wantRows) {
		t.Errorf("got different rows")
	}
}

func TestLargeCSV(t *testing.T) {
	const lines = 1000000

	type Request struct {
	}

	getHandler := func(ctx context.Context, req *Request) (res *api2.CsvResponse, err error) {
		rows := make(chan []string)
		go func() {
			defer close(rows)
			for i := 0; i < lines; i++ {
				rows <- []string{"Alice", "31"}
			}
		}()
		return &api2.CsvResponse{
			HttpCode:  http.StatusOK,
			CsvHeader: []string{"Name", "Age"},
			Rows:      rows,
		}, nil
	}

	routes := []api2.Route{
		{Method: http.MethodGet, Path: "/csv", Handler: getHandler, Transport: api2.CsvTransport},
	}

	mux := http.NewServeMux()
	api2.BindRoutes(mux, routes)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	client := api2.NewClient(routes, server.URL)

	req := &Request{}
	res := &api2.CsvResponse{
		Rows: make(chan []string),
	}

	var wg sync.WaitGroup

	count := 0
	wg.Add(1)
	go func() {
		defer wg.Done()
		for range res.Rows {
			count++
		}
	}()

	err := client.Call(context.Background(), res, req)
	if err != nil {
		t.Errorf("request failed: %v.", err)
	}

	wg.Wait()

	if count != lines {
		t.Errorf("got %d lines, want %d", count, lines)
	}
}

func TestErrorCSV(t *testing.T) {
	wantMessage := "file not found"

	type Request struct {
	}

	getHandler := func(ctx context.Context, req *Request) (res *api2.CsvResponse, err error) {
		return nil, errors.NotFound(wantMessage)
	}

	routes := []api2.Route{
		{Method: http.MethodGet, Path: "/csv", Handler: getHandler, Transport: api2.CsvTransport},
	}

	mux := http.NewServeMux()
	api2.BindRoutes(mux, routes)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	client := api2.NewClient(routes, server.URL)

	req := &Request{}
	res := &api2.CsvResponse{
		Rows: make(chan []string),
	}

	err := client.Call(context.Background(), res, req)
	if err == nil {
		t.Fatalf("did not get an expected error")
	}

	code := err.(api2.HttpError).HttpCode()
	if code != http.StatusNotFound {
		t.Fatalf("wrong code: want %d, got %d", http.StatusNotFound, code)
	}

	if err.Error() != wantMessage {
		t.Fatalf("wrong message: want %q, got %q", wantMessage, err.Error())
	}
}
