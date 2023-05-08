package api2

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/starius/api2"
	"github.com/starius/api2/errors"
	"github.com/stretchr/testify/require"
)

type TooManyKeysError struct {
	Keys int `json:"keys"`
}

func (e TooManyKeysError) Error() string {
	return fmt.Sprintf("too many keys: %d", e.Keys)
}

func TestHuman(t *testing.T) {
	type HelloRequest struct {
		Foo map[string]int `json:"foo"`
	}
	type HelloResponse struct {
		RevFoo map[int]string `json:"bar"`
	}

	helloHandler := func(ctx context.Context, req *HelloRequest) (res *HelloResponse, err error) {
		if len(req.Foo) > 5 {
			return nil, errors.ResourceExhausted("too large request: %w", TooManyKeysError{Keys: len(req.Foo)})
		}
		revFoo := make(map[int]string, len(req.Foo))
		for key, value := range req.Foo {
			if _, has := revFoo[value]; has {
				return nil, errors.InvalidArgument("duplicate value: %d", value)
			}
			revFoo[value] = key
		}
		return &HelloResponse{RevFoo: revFoo}, nil
	}

	routes := []api2.Route{
		{
			Method:  http.MethodPost,
			Path:    "/hello",
			Handler: helloHandler,
			Transport: &api2.JsonTransport{
				Errors: map[string]error{
					"TooManyKeysError": TooManyKeysError{},
				},
			},
		},
	}

	muxHuman := http.NewServeMux()
	api2.BindRoutes(muxHuman, routes, api2.HumanJSON(true))
	muxNoHuman := http.NewServeMux()
	api2.BindRoutes(muxNoHuman, routes)

	t.Run("make sure human server works with non-human api2 client", func(t *testing.T) {
		server := httptest.NewServer(muxHuman)
		t.Cleanup(server.Close)
		client := api2.NewClient(routes, server.URL)

		ctx := context.Background()

		var res HelloResponse
		require.NoError(t, client.Call(ctx, &res, &HelloRequest{
			Foo: map[string]int{
				"a": 1,
				"b": 2,
			},
		}))
		wantRes := HelloResponse{
			RevFoo: map[int]string{
				1: "a",
				2: "b",
			},
		}
		require.Equal(t, wantRes, res)

		require.Error(t, client.Call(ctx, &res, &HelloRequest{
			Foo: map[string]int{
				"a": 1,
				"b": 1,
			},
		}))
	})

	t.Run("make sure non-human server works with human api2 client", func(t *testing.T) {
		server := httptest.NewServer(muxNoHuman)
		t.Cleanup(server.Close)
		client := api2.NewClient(routes, server.URL, api2.HumanJSON(true))

		ctx := context.Background()

		var res HelloResponse
		require.NoError(t, client.Call(ctx, &res, &HelloRequest{
			Foo: map[string]int{
				"a": 1,
				"b": 2,
			},
		}))
		wantRes := HelloResponse{
			RevFoo: map[int]string{
				1: "a",
				2: "b",
			},
		}
		require.Equal(t, wantRes, res)

		require.Error(t, client.Call(ctx, &res, &HelloRequest{
			Foo: map[string]int{
				"a": 1,
				"b": 1,
			},
		}))
	})

	t.Run("test server", func(t *testing.T) {
		cases := []struct {
			name            string
			method          string
			reqBody         string
			wantCode        int
			wantHumanBody   string
			wantNoHumanBody string
		}{
			{
				name:   "success",
				method: "POST",
				reqBody: `{
					"foo": {
						"a": 1,
						"b": 2
					}
				}`,
				wantCode:        http.StatusOK,
				wantHumanBody:   "{\n  \"bar\": {\n    \"1\": \"a\",\n    \"2\": \"b\"\n  }\n}\n",
				wantNoHumanBody: `{"bar":{"1":"a","2":"b"}}` + "\n",
			},
			{
				name:   "error",
				method: "POST",
				reqBody: `{
					"foo": {
						"a": 1,
						"b": 1
					}
				}`,
				wantCode:        http.StatusBadRequest,
				wantHumanBody:   "{\n  \"error\": \"duplicate value: 1\"\n}\n",
				wantNoHumanBody: `{"error":"duplicate value: 1"}` + "\n",
			},
			{
				name:   "wrapped error",
				method: "POST",
				reqBody: `{
					"foo": {
						"a": 1,
						"b": 2,
						"c": 3,
						"d": 4,
						"e": 5,
						"f": 6
					}
				}`,
				wantCode:        http.StatusTooManyRequests,
				wantHumanBody:   "{\n  \"error\": \"too large request: too many keys: 6\",\n  \"detail\": {\n    \"keys\": 6\n  },\n  \"code\": \"TooManyKeysError\"\n}\n",
				wantNoHumanBody: `{"error":"too large request: too many keys: 6","detail":{"keys":6},"code":"TooManyKeysError"}` + "\n",
			},
			{
				name:            "bad method",
				method:          "GET",
				reqBody:         `{}`,
				wantCode:        http.StatusMethodNotAllowed,
				wantHumanBody:   "{\n  \"error\": \"unsupported method: GET\"\n}\n",
				wantNoHumanBody: `{"error":"unsupported method: GET"}` + "\n",
			},
			{
				name:            "bad JSON",
				method:          "POST",
				reqBody:         `{`,
				wantCode:        http.StatusBadRequest,
				wantHumanBody:   "{\n  \"error\": \"failed to parse request: unexpected EOF\"\n}\n",
				wantNoHumanBody: `{"error":"failed to parse request: unexpected EOF"}` + "\n",
			},
		}

		for _, tc := range cases {
			tc := tc
			t.Run(tc.name, func(t *testing.T) {
				t.Run("human server", func(t *testing.T) {
					req := httptest.NewRequest(tc.method, "/hello", bytes.NewBufferString(tc.reqBody))
					res := httptest.NewRecorder()
					muxHuman.ServeHTTP(res, req)
					require.Equal(t, tc.wantCode, res.Code)
					require.Equal(t, tc.wantHumanBody, res.Body.String())
				})
				t.Run("human client", func(t *testing.T) {
					req := httptest.NewRequest(tc.method, "/hello?human=on", bytes.NewBufferString(tc.reqBody))
					res := httptest.NewRecorder()
					muxNoHuman.ServeHTTP(res, req)
					require.Equal(t, tc.wantCode, res.Code)
					require.Equal(t, tc.wantHumanBody, res.Body.String())
				})
				t.Run("not human", func(t *testing.T) {
					req := httptest.NewRequest(tc.method, "/hello", bytes.NewBufferString(tc.reqBody))
					res := httptest.NewRecorder()
					muxNoHuman.ServeHTTP(res, req)
					require.Equal(t, tc.wantCode, res.Code)
					require.Equal(t, tc.wantNoHumanBody, res.Body.String())
				})
			})
		}
	})

	t.Run("test client", func(t *testing.T) {
		var buf bytes.Buffer
		var serverErr error
		handler := func(w http.ResponseWriter, r *http.Request) {
			_, serverErr = io.Copy(&buf, r.Body)
			if serverErr != nil {
				return
			}
			_, serverErr = w.Write([]byte(`{"bar": {"1": "a", "2": "b"}}`))
		}
		server := httptest.NewServer(http.HandlerFunc(handler))
		t.Cleanup(server.Close)

		client := api2.NewClient(routes, server.URL, api2.HumanJSON(true))

		ctx := context.Background()

		var res HelloResponse
		require.NoError(t, client.Call(ctx, &res, &HelloRequest{
			Foo: map[string]int{
				"a": 1,
				"b": 2,
			},
		}))

		require.NoError(t, serverErr)
		wantClientBody := "{\n  \"foo\": {\n    \"a\": 1,\n    \"b\": 2\n  }\n}\n"
		require.Equal(t, wantClientBody, buf.String())

		wantRes := HelloResponse{
			RevFoo: map[int]string{
				1: "a",
				2: "b",
			},
		}
		require.Equal(t, wantRes, res)
	})

	t.Run("test client - control", func(t *testing.T) {
		var buf bytes.Buffer
		var serverErr error
		handler := func(w http.ResponseWriter, r *http.Request) {
			_, serverErr = io.Copy(&buf, r.Body)
			if serverErr != nil {
				return
			}
			_, serverErr = w.Write([]byte(`{"bar": {"1": "a", "2": "b"}}`))
		}
		server := httptest.NewServer(http.HandlerFunc(handler))
		t.Cleanup(server.Close)

		client := api2.NewClient(routes, server.URL)

		ctx := context.Background()

		var res HelloResponse
		require.NoError(t, client.Call(ctx, &res, &HelloRequest{
			Foo: map[string]int{
				"a": 1,
				"b": 2,
			},
		}))

		require.NoError(t, serverErr)
		wantClientBody := `{"foo":{"a":1,"b":2}}` + "\n"
		require.Equal(t, wantClientBody, buf.String())

		wantRes := HelloResponse{
			RevFoo: map[int]string{
				1: "a",
				2: "b",
			},
		}
		require.Equal(t, wantRes, res)
	})
}
