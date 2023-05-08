package debugclient

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"sync/atomic"

	"moul.io/http2curl"
)

type HttpClient interface {
	Do(req *http.Request) (*http.Response, error)
	CloseIdleConnections()
}

type DebugClient struct {
	impl HttpClient
	log  io.Writer
	n    uint64
}

func New(impl HttpClient, log io.Writer) (*DebugClient, error) {
	return &DebugClient{
		impl: impl,
		log:  log,
	}, nil
}

func (c *DebugClient) Do(req *http.Request) (*http.Response, error) {
	n := atomic.AddUint64(&c.n, 1)

	curl, err := http2curl.GetCurlCommand(req)
	if err != nil {
		return nil, fmt.Errorf("http2curl.GetCurlCommand failed for %d: %w", n, err)
	}
	if _, err = fmt.Fprintf(c.log, "=== client request %d ===\n$ %s\n=== end of client request %d ===\n", n, curl, n); err != nil {
		return nil, fmt.Errorf("fmt.Fprintf(request) failed for %d: %w", n, err)
	}

	res, err := c.impl.Do(req)
	if err != nil {
		return nil, err
	}

	resDump, err := httputil.DumpResponse(res, true)
	if err != nil {
		return nil, fmt.Errorf("httputil.DumpResponse failed for %d: %w", n, err)
	}
	if _, err = fmt.Fprintf(c.log, "=== server response %d ===\n%s\n=== end of server response %d ===\n", n, string(resDump), n); err != nil {
		return nil, fmt.Errorf("fmt.Fprintf(response) failed for %d: %w", n, err)
	}

	return res, nil
}

func (c *DebugClient) CloseIdleConnections() {
	c.impl.CloseIdleConnections()
}
