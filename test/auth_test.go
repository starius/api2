package api2

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/starius/api2"
)

func TestAuth(t *testing.T) {
	type HelloRequest struct {
	}
	type HelloResponse struct {
		Ok bool `json:"ok"`
	}

	helloHandler := func(ctx context.Context, req *HelloRequest) (res *HelloResponse, err error) {
		return &HelloResponse{
			Ok: true,
		}, nil
	}

	type EchoRequest struct {
		Foo int `json:"foo"`
	}
	type EchoResponse struct {
		Foo int `json:"foo"`
	}

	echoHandler := func(ctx context.Context, req *EchoRequest) (res *EchoResponse, err error) {
		return &EchoResponse{
			Foo: req.Foo,
		}, nil
	}

	const (
		publicKey           = "public"
		allowedUsersKey     = "allowed-users"
		authorizationHeader = "Authorization"
	)

	routes := []api2.Route{
		// Public method.
		{
			Method:  http.MethodPost,
			Path:    "/hello",
			Handler: helloHandler,
			Meta: map[string]interface{}{
				publicKey: true,
			},
		},

		// Protected method.
		{
			Method:  http.MethodPost,
			Path:    "/echo",
			Handler: echoHandler,
			Meta: map[string]interface{}{
				allowedUsersKey: []string{"alice", "bob"},
			},
		},
	}

	// For the test, we use basic auth with password = username.
	makeAuth := func(username string) string {
		// Reuse code of http.Request for generating basic auth header.
		fakeRequest, err := http.NewRequest("GET", "http://example.com", nil)
		if err != nil {
			panic(err)
		}
		fakeRequest.SetBasicAuth(username, username)
		return fakeRequest.Header.Get(authorizationHeader)
	}
	checkAuth := func(route *api2.Route, r *http.Request) bool {
		isPublic, has := route.Meta[publicKey]
		if has && isPublic.(bool) == true {
			return true
		}
		allowedUsers, has := route.Meta[allowedUsersKey]
		if has {
			username, password, ok := r.BasicAuth()
			if !ok {
				return false
			}
			// Pretend we check the password.
			if password != username {
				return false
			}
			for _, goodUser := range allowedUsers.([]string) {
				if username == goodUser {
					return true
				}
			}
		}
		return false
	}

	matcher := api2.GetMatcher(routes)

	mux := http.NewServeMux()
	api2.BindRoutes(mux, routes)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		route, has := matcher(r)
		if !has {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if !checkAuth(route, r) {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		mux.ServeHTTP(w, r)
	}))
	t.Cleanup(server.Close)

	ctx := context.Background()

	t.Run("alice", func(t *testing.T) {
		client := api2.NewClient(routes, server.URL, api2.AuthorizationHeader(makeAuth("alice")))

		helloRes := &HelloResponse{}
		err := client.Call(ctx, helloRes, &HelloRequest{})
		if err != nil {
			t.Errorf("Hello failed for Alice: %v.", err)
		}

		echoRes := &EchoResponse{}
		err = client.Call(ctx, echoRes, &EchoRequest{
			Foo: 10,
		})
		if err != nil {
			t.Errorf("Foo failed for Alice: %v.", err)
		}
		if echoRes.Foo != 10 {
			t.Errorf("Foo returned %d, want %d.", echoRes.Foo, 10)
		}
	})

	t.Run("eve", func(t *testing.T) {
		client := api2.NewClient(routes, server.URL, api2.AuthorizationHeader(makeAuth("eve")))

		helloRes := &HelloResponse{}
		err := client.Call(ctx, helloRes, &HelloRequest{})
		if err != nil {
			t.Errorf("Hello failed for Eve: %v.", err)
		}

		echoRes := &EchoResponse{}
		err = client.Call(ctx, echoRes, &EchoRequest{
			Foo: 10,
		})
		if err == nil {
			t.Errorf("Foo did not fail for Eve.")
		}
	})
}
