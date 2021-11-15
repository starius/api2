package api2

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
)

// Client is used on client-side to call remote methods provided by the API.
type Client struct {
	routeMap      map[signature]Route
	client        *http.Client
	baseURL       string
	errorf        func(format string, args ...interface{})
	authorization string
	maxBody       int64
}

type signature struct {
	request  reflect.Type
	response reflect.Type
}

// NewClient creates new instance of client.
//
// The list of routes must provide all routes that this client is aware of.
// Paths from the table of routes are appended to baseURL to generate final
// URL used by HTTP client.
// All pairs of (request type, response type) must be unique in the table
// of routes.
func NewClient(routes []Route, baseURL string, opts ...Option) *Client {
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	routeMap := make(map[signature]Route, len(routes))
	for _, route := range routes {
		handler := route.Handler
		if m, ok := handler.(*interfaceMethod); ok {
			handler = m.Func()
		}
		handlerType := reflect.TypeOf(handler)
		validateHandler(handlerType)
		key := signature{
			request:  handlerType.In(1),
			response: handlerType.Out(0),
		}
		if _, has := routeMap[key]; has {
			panic(fmt.Sprintf("Already has a handler with signature %v.", key))
		}
		routeMap[key] = route
	}

	config := NewDefaultConfig()
	for _, opt := range opts {
		opt(config)
	}
	if config.client != nil {
		client = config.client
	}

	return &Client{
		routeMap:      routeMap,
		client:        client,
		baseURL:       baseURL,
		errorf:        config.errorf,
		authorization: config.authorization,
		maxBody:       config.maxBody,
	}
}

// Call calls remote method deduced by request and response types.
// Both request and response must be pointers to structs.
// The method must be called on exactly the same types as the
// corresponding method of a service.
func (c *Client) Call(ctx context.Context, response, request interface{}) error {
	key := signature{
		request:  reflect.TypeOf(request),
		response: reflect.TypeOf(response),
	}
	route, has := c.routeMap[key]
	if !has {
		panic(fmt.Sprintf("No registered method with signature %v %v.", key.request, key.response))
	}

	t := route.Transport
	if t == nil {
		t = DefaultTransport
	}

	url := c.baseURL + route.Path

	req, err := t.EncodeRequest(ctx, route.Method, url, request)
	if err != nil {
		return fmt.Errorf("failed to encode request: %w", err)
	}

	if c.authorization != "" {
		req.Header.Set("Authorization", c.authorization)
	}

	res, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	res.Body = http.MaxBytesReader(nil, res.Body, c.maxBody)
	defer func() {
		if err := res.Body.Close(); err != nil {
			c.errorf("failed to close resource: %v", err)
		}
	}()

	// Handle all 2xx responses as success.
	if 200 <= res.StatusCode && res.StatusCode < 300 {
		return t.DecodeResponse(req.Context(), res, response)
	} else {
		return t.DecodeError(req.Context(), res)
	}
}

func (c *Client) Close() error {
	c.client.CloseIdleConnections()
	return nil
}
