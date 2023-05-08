package debugclient

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/starius/api2"
	"github.com/stretchr/testify/require"
)

func TestDebugClient(t *testing.T) {
	type HelloRequest struct {
		Foo int `json:"foo"`
	}
	type HelloResponse struct {
		Bar int `json:"bar"`
	}

	helloHandler := func(ctx context.Context, req *HelloRequest) (res *HelloResponse, err error) {
		return &HelloResponse{Bar: req.Foo + 1}, nil
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

	var log bytes.Buffer

	debugClient, err := New(http.DefaultClient, &log)
	require.NoError(t, err)
	client := api2.NewClient(routes, server.URL, api2.CustomClient(debugClient))

	ctx := context.Background()

	helloRes := &HelloResponse{}
	require.NoError(t, client.Call(ctx, helloRes, &HelloRequest{Foo: 123}))
	require.Equal(t, 124, helloRes.Bar)

	gotLog := log.String()

	wantLog := "=== client request 1 ===\n$ curl -X 'POST' -d '{\"foo\":123}\n' -H 'Accept: application/json' -H 'Content-Type: application/json; charset=UTF-8' 'http://127.0.0.1:46821/hello'\n=== end of client request 1 ===\n=== server response 1 ===\nHTTP/1.1 200 OK\r\nContent-Length: 12\r\nContent-Type: application/json; charset=UTF-8\r\nDate: Mon, 08 May 2023 07:20:28 GMT\r\n\r\n{\"bar\":124}\n\n=== end of server response 1 ===\n"

	// Replace non-deterministic elements.
	re := regexp.MustCompile(`http://127.0.0.1:[0-9]+/hello|Date: [^\r\n]+\r\n`)

	gotLog = re.ReplaceAllString(gotLog, "X")
	wantLog = re.ReplaceAllString(wantLog, "X")

	require.Equal(t, wantLog, gotLog)

}
