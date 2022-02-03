package closingclient

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/starius/api2"
)

func TestClosingClient(t *testing.T) {
	type HelloRequest struct {
		Sleep time.Duration `json:"sleep"`
	}
	type HelloResponse struct {
	}

	helloHandler := func(ctx context.Context, req *HelloRequest) (res *HelloResponse, err error) {
		time.Sleep(req.Sleep)
		return &HelloResponse{}, nil
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

	ctx := context.Background()

	t.Run("normal client", func(t *testing.T) {
		client := api2.NewClient(routes, server.URL)

		var errCall, errClose error
		var wg sync.WaitGroup

		t1 := time.Now()

		wg.Add(1)
		go func() {
			defer wg.Done()
			helloRes := &HelloResponse{}
			errCall = client.Call(ctx, helloRes, &HelloRequest{Sleep: time.Second})
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			time.Sleep(100 * time.Millisecond)

			t1 := time.Now()
			errClose = client.Close()
			spent := time.Since(t1)

			if spent > 10*time.Millisecond {
				errClose = fmt.Errorf("Expected Close to spend 0.01s or less, but spent %s.", spent)
			}
		}()

		wg.Wait()

		if errCall != nil {
			t.Errorf("Call failed: %v.", errCall)
		}
		if errClose != nil {
			t.Errorf("Close failed: %v.", errClose)
		}

		spent := time.Since(t1)
		if spent < time.Second {
			t.Errorf("In normal client expected to spend 1s or more, but spent %s.", spent)
		}
	})

	t.Run("closing client", func(t *testing.T) {
		cc, err := New(http.DefaultClient)
		if err != nil {
			t.Fatal(err)
		}
		client := api2.NewClient(routes, server.URL, api2.CustomClient(cc))

		var errCall, errClose error
		var wg sync.WaitGroup

		t1 := time.Now()

		wg.Add(1)
		go func() {
			defer wg.Done()
			helloRes := &HelloResponse{}
			errCall = client.Call(ctx, helloRes, &HelloRequest{Sleep: time.Second})
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			time.Sleep(100 * time.Millisecond)

			t1 := time.Now()
			errClose = client.Close()
			spent := time.Since(t1)

			if spent > 10*time.Millisecond {
				errClose = fmt.Errorf("Expected Close to spend 0.01s or less, but spent %s.", spent)
			}
		}()

		wg.Wait()

		if errCall == nil {
			t.Errorf("Call not failed.")
		}
		if errClose != nil {
			t.Errorf("Close failed: %v.", errClose)
		}

		spent := time.Since(t1)
		if spent > time.Second/2 {
			t.Errorf("In closing client expected to spend 0.5s or less, but spent %s.", spent)
		}
	})

	t.Run("closing client, many parallel requests", func(t *testing.T) {
		cc, err := New(http.DefaultClient)
		if err != nil {
			t.Fatal(err)
		}
		client := api2.NewClient(routes, server.URL, api2.CustomClient(cc))

		var errClose1, errClose2, errClose3 error
		var wg sync.WaitGroup

		t1 := time.Now()

		n := 50
		wg.Add(n)
		for i := 0; i < n; i++ {
			i := i
			time.AfterFunc(time.Duration(i)*time.Millisecond*10, func() {
				defer wg.Done()
				helloRes := &HelloResponse{}
				err := client.Call(ctx, helloRes, &HelloRequest{Sleep: 200 * time.Millisecond})
				t.Log(i, err)
			})
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			time.Sleep(250 * time.Millisecond)

			cc.mu.Lock()
			ncancels := len(cc.cancels)
			cc.mu.Unlock()
			fmt.Println(ncancels)
			if ncancels >= 25 {
				errClose1 = fmt.Errorf("Expected to have < 25 cancels in the map, got %d", ncancels)
			}

			t1 := time.Now()
			errClose2 = client.Close()
			spent := time.Since(t1)

			if spent > 10*time.Millisecond {
				errClose3 = fmt.Errorf("Expected Close to spend 0.01s or less, but spent %s.", spent)
			}
		}()

		wg.Wait()

		if errClose1 != nil {
			t.Errorf("Close failed: %v.", errClose1)
		}

		if errClose2 != nil {
			t.Errorf("Close failed: %v.", errClose2)
		}

		if errClose3 != nil {
			t.Errorf("Close failed: %v.", errClose3)
		}

		spent := time.Since(t1)
		if spent > 600*time.Millisecond {
			t.Errorf("Expected to spend 0.6s or less, but spent %s.", spent)
		}
	})
}
