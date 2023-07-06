package closingclient

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
)

type HttpClient interface {
	Do(req *http.Request) (*http.Response, error)
	CloseIdleConnections()
}

type ClosingClient struct {
	impl HttpClient

	mu            sync.Mutex
	closing       bool
	cancels       map[uint64]func()
	lastCancelKey uint64

	wg sync.WaitGroup
}

func New(impl HttpClient) (*ClosingClient, error) {
	return &ClosingClient{
		impl:    impl,
		cancels: make(map[uint64]func()),
	}, nil
}

func (c *ClosingClient) Do(req *http.Request) (*http.Response, error) {
	ctx, cancel := context.WithCancel(req.Context())

	c.mu.Lock()
	if c.closing {
		c.mu.Unlock()
		cancel()
		return nil, fmt.Errorf("api2 client is closing")
	}

	// Add(1) and Wait() must not be called in parallel.
	// Call Add(1) under mutex protecting c.closing.
	c.wg.Add(1)
	defer c.wg.Done()

	key := c.lastCancelKey
	c.lastCancelKey++
	c.cancels[key] = cancel
	c.mu.Unlock()

	defer func() {
		c.mu.Lock()
		defer c.mu.Unlock()
		delete(c.cancels, key)
	}()

	return c.impl.Do(req.Clone(ctx))
}

func (c *ClosingClient) CloseIdleConnections() {
	c.impl.CloseIdleConnections()
}

func (c *ClosingClient) Close() error {
	c.mu.Lock()
	if !c.closing {
		c.closing = true
		// Close active connections.
		for _, cancel := range c.cancels {
			cancel()
		}
		c.cancels = nil
	}
	c.mu.Unlock()

	c.impl.CloseIdleConnections()

	// Add(1) and Wait() must not be called in parallel.
	// By this point, c.closing=true, so if there are any calls
	// of Do() between mu.Unlock above and this line, they
	// won't result in Add(1).
	c.wg.Wait()

	if closer, ok := c.impl.(io.Closer); ok {
		if err := closer.Close(); err != nil {
			return err
		}
	}

	return nil
}
