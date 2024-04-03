package api2

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/starius/api2"
	"github.com/stretchr/testify/require"
)

type closeRecorder struct {
	r      io.Reader
	closed bool
}

func (c *closeRecorder) Read(p []byte) (n int, err error) {
	return c.r.Read(p)
}

func (c *closeRecorder) Close() error {
	c.closed = true
	return nil
}

func TestStream(t *testing.T) {
	type Request struct {
		Body io.ReadCloser `use_as_body:"true" is_stream:"true"`
	}

	type Response struct {
		Body io.ReadCloser `use_as_body:"true" is_stream:"true"`
	}

	xorHandler := func(ctx context.Context, req *Request) (res *Response, err error) {
		r, w := io.Pipe()
		go func() {
			// Read byte by byte, xor with 0x42 and write to output stream.
			buf := make([]byte, 1)
			for {
				n, err := req.Body.Read(buf)
				if err != nil && err != io.EOF {
					panic(err)
				}
				if n == 0 {
					if err == io.EOF {
						break
					}
					continue
				}
				if n > 1 {
					panic(fmt.Errorf("expected to get <= 1 bytes, got %d", n))
				}
				buf[0] ^= 0x42
				n, err2 := w.Write(buf)
				if err2 != nil {
					panic(err2)
				}
				if n != 1 {
					panic(fmt.Errorf("expected to get n = 1 byte, got %d", n))
				}
				if err == io.EOF {
					break
				}
			}
			if err := w.Close(); err != nil {
				panic(err)
			}
		}()
		return &Response{
			Body: r,
		}, nil
	}

	routes := []api2.Route{
		{Method: http.MethodGet, Path: "/stream", Handler: xorHandler},
	}

	mux := http.NewServeMux()
	api2.BindRoutes(mux, routes)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	client := api2.NewClient(routes, server.URL)

	rc := &closeRecorder{r: bytes.NewReader([]byte{0x00, 0xFF, 0x11})}

	req := &Request{
		Body: rc,
	}
	res := &Response{}

	err := client.Call(context.Background(), res, req)
	if err != nil {
		t.Errorf("request failed: %v.", err)
	}

	if !rc.closed {
		t.Errorf("expected the library to close the request reader")
	}

	buf, err := io.ReadAll(res.Body)
	if err != nil {
		t.Errorf("failed to read response body: %v.", err)
	}
	want := []byte{0x42, 0xFF ^ 0x42, 0x11 ^ 0x42}
	if !bytes.Equal(buf, want) {
		t.Errorf("got buf=%v, want %v", buf, want)
	}
}

func TestStreamRequestOnly(t *testing.T) {
	type Request struct {
		Body io.ReadCloser `use_as_body:"true" is_stream:"true"`
	}

	type Response struct {
		Foo string `json:"foo"`
	}

	handler := func(ctx context.Context, req *Request) (res *Response, err error) {
		data, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		return &Response{
			Foo: string(data),
		}, nil
	}

	routes := []api2.Route{
		{Method: http.MethodGet, Path: "/stream", Handler: handler},
	}

	mux := http.NewServeMux()
	api2.BindRoutes(mux, routes)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	client := api2.NewClient(routes, server.URL)

	rc := &closeRecorder{r: bytes.NewReader([]byte("Hello"))}

	req := &Request{
		Body: rc,
	}
	res := &Response{}

	err := client.Call(context.Background(), res, req)
	if err != nil {
		t.Errorf("request failed: %v.", err)
	}

	if !rc.closed {
		t.Errorf("expected the library to close the request reader")
	}

	require.Equal(t, "Hello", res.Foo)
}

func TestStreamResponseOnly(t *testing.T) {
	type Request struct {
		Foo string `json:"foo"`
	}

	type Response struct {
		Body io.ReadCloser `use_as_body:"true" is_stream:"true"`
	}

	handler := func(ctx context.Context, req *Request) (res *Response, err error) {
		return &Response{
			Body: io.NopCloser(bytes.NewReader([]byte(req.Foo))),
		}, nil
	}

	routes := []api2.Route{
		{Method: http.MethodGet, Path: "/stream", Handler: handler},
	}

	mux := http.NewServeMux()
	api2.BindRoutes(mux, routes)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	client := api2.NewClient(routes, server.URL)

	req := &Request{
		Foo: "Hello",
	}
	res := &Response{}

	err := client.Call(context.Background(), res, req)
	if err != nil {
		t.Errorf("request failed: %v.", err)
	}

	buf, err := io.ReadAll(res.Body)
	if err != nil {
		t.Errorf("failed to read response body: %v.", err)
	}
	want := []byte("Hello")
	if !bytes.Equal(buf, want) {
		t.Errorf("got buf=%v, want %v", buf, want)
	}
}

func TestStreamNoBodyErrors(t *testing.T) {
	type Request struct {
		Body            io.ReadCloser `use_as_body:"true" is_stream:"true"`
		SetResponseBody bool          `header:"set-response-body"`
	}

	type Response struct {
		Body io.ReadCloser `use_as_body:"true" is_stream:"true"`
	}

	handler := func(ctx context.Context, req *Request) (res *Response, err error) {
		if req.SetResponseBody {
			return &Response{
				Body: io.NopCloser(bytes.NewReader([]byte("123"))),
			}, nil
		} else {
			return &Response{
				Body: nil,
			}, nil
		}
	}

	routes := []api2.Route{
		{Method: http.MethodGet, Path: "/stream", Handler: handler},
	}

	mux := http.NewServeMux()
	api2.BindRoutes(mux, routes)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	client := api2.NewClient(routes, server.URL)

	t.Run("request without body", func(t *testing.T) {
		req := &Request{
			Body:            nil,
			SetResponseBody: true,
		}
		res := &Response{}
		err := client.Call(context.Background(), res, req)
		require.NoError(t, err)
		_, err = io.Copy(io.Discard, res.Body)
		require.NoError(t, err)
		require.NoError(t, res.Body.Close())
	})

	t.Run("response without body", func(t *testing.T) {
		req := &Request{
			Body:            io.NopCloser(bytes.NewReader([]byte("123"))),
			SetResponseBody: false,
		}
		res := &Response{}
		err := client.Call(context.Background(), res, req)
		require.NoError(t, err)
		_, err = io.Copy(io.Discard, res.Body)
		require.NoError(t, err)
		require.NoError(t, res.Body.Close())
	})

	t.Run("request and response without body", func(t *testing.T) {
		req := &Request{
			Body:            nil,
			SetResponseBody: false,
		}
		res := &Response{}
		err := client.Call(context.Background(), res, req)
		require.NoError(t, err)
		_, err = io.Copy(io.Discard, res.Body)
		require.NoError(t, err)
		require.NoError(t, res.Body.Close())
	})
}

func BenchmarkStream(b *testing.B) {
	type Request struct {
		Body io.ReadCloser `use_as_body:"true" is_stream:"true"`
	}

	type Response struct {
		Body io.ReadCloser `use_as_body:"true" is_stream:"true"`
	}

	const mb = 1024 * 1024
	b.SetBytes(mb)

	nBytes := int64(b.N * mb)

	handler := func(ctx context.Context, req *Request) (res *Response, err error) {
		// Read whole request body.
		n, err := io.Copy(io.Discard, req.Body)
		if err != nil {
			panic(err)
		}
		if n != nBytes {
			panic(fmt.Errorf("expected to get n = %d, got %d", nBytes, n))
		}

		return &Response{
			Body: io.NopCloser(&io.LimitedReader{
				R: rand.New(rand.NewSource(42)),
				N: nBytes,
			}),
		}, nil
	}

	routes := []api2.Route{
		{Method: http.MethodGet, Path: "/stream", Handler: handler},
	}

	mux := http.NewServeMux()
	api2.BindRoutes(mux, routes, api2.MaxBody(nBytes))
	server := httptest.NewServer(mux)
	b.Cleanup(server.Close)

	client := api2.NewClient(routes, server.URL, api2.MaxBody(nBytes))

	req := &Request{
		Body: io.NopCloser(&io.LimitedReader{
			R: rand.New(rand.NewSource(42)),
			N: nBytes,
		}),
	}
	res := &Response{}

	err := client.Call(context.Background(), res, req)
	if err != nil {
		b.Errorf("request failed: %v.", err)
	}

	n, err := io.Copy(io.Discard, res.Body)
	if err != nil {
		panic(err)
	}
	if n != nBytes {
		panic(fmt.Errorf("expected to get n = %d, got %d", nBytes, n))
	}
}
