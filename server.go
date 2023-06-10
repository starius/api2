package api2

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
)

type errorMessage struct {
	Error  string          `json:"error"`
	Detail json.RawMessage `json:"detail,omitempty"`
	Code   string          `json:"code,omitempty"`
}

type Router interface {
	HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request))
}

func jsonError(w http.ResponseWriter, human bool, code int, format string, args ...interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	errmsg := fmt.Sprintf(format, args...)
	return newEncoder(w, human).Encode(errorMessage{Error: errmsg})
}

// BindRoutes adds handlers of routes to http.ServeMux.
func BindRoutes(mux Router, routes []Route, opts ...Option) {
	config := NewDefaultConfig()
	for _, opt := range opts {
		opt(config)
	}
	errorf := config.errorf
	human := config.human

	path2routes := make(map[string][]Route)
	for _, route := range routes {
		path := cutUrlParams(route.Path)
		path2routes[path] = append(path2routes[path], route)
	}

	for path, routes := range path2routes {
		method2routes := make(map[string][]Route, len(routes))
		for _, route := range routes {
			method2routes[route.Method] = append(method2routes[route.Method], route)
		}
		method2handler := make(map[string]http.HandlerFunc, len(routes))
		for method, routes := range method2routes {
			method2handler[method] = newHTTPMethodHandler(routes, human, errorf)
		}

		mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			// Calling FormValue before parsing JSON "eats" r.Body if Content-Type is
			// application/x-www-form-urlencoded. This happens in curl for me.
			human2 := human || r.FormValue("human") != ""
			if human2 {
				r = r.WithContext(context.WithValue(r.Context(), humanType{}, true))
			}
			handler, has := method2handler[r.Method]
			if !has {
				if err := jsonError(w, human2, http.StatusMethodNotAllowed, "unsupported method: %v", r.Method); err != nil {
					errorf("%s handler failed to send MethodNotAllowed error to client: %v", r.URL.Path, err)
				}
				return
			}
			r.Body = http.MaxBytesReader(w, r.Body, config.maxBody)
			handler(w, r)
		})
	}
}

// GetMatcher returns a function converting http.Request to Route.
func GetMatcher(routes []Route) func(*http.Request) (*Route, bool) {
	path2method2route := make(map[string]map[string]*Route)
	for _, route := range routes {
		route := route
		method2route, has := path2method2route[route.Path]
		if !has {
			method2route = make(map[string]*Route)
			path2method2route[route.Path] = method2route
		}
		method2route[route.Method] = &route
	}

	// Use mux to detect route.Path from http.Request.
	mux := http.NewServeMux()
	BindRoutes(mux, routes)

	return func(r *http.Request) (*Route, bool) {
		_, path := mux.Handler(r)
		method2route, has := path2method2route[path]
		if !has {
			return nil, false
		}
		route, has := method2route[r.Method]
		return route, has
	}
}

func newHTTPMethodHandler(routes []Route, human bool, errorf func(format string, args ...interface{})) http.HandlerFunc {
	if len(routes) == 1 && len(findUrlKeys(routes[0].Path)) == 0 {
		// Single handler without URL parameters.
		return newHTTPHandler(routes[0], human, errorf)
	}
	paths := make([]string, 0, len(routes))
	handlers := make([]http.HandlerFunc, 0, len(routes))
	for _, route := range routes {
		paths = append(paths, route.Path)
		handlers = append(handlers, newHTTPHandler(route, human, errorf))
	}
	c := newPathClassifier(paths)

	return func(w http.ResponseWriter, r *http.Request) {
		index, param2value := c.Classify(r.URL.Path)
		if index == -1 {
			// Calling FormValue before parsing JSON "eats" r.Body if Content-Type is
			// application/x-www-form-urlencoded. This happens in curl for me.
			human2 := human || r.FormValue("human") != ""
			if err := jsonError(w, human2, http.StatusNotFound, "failed to find route by path"); err != nil {
				errorf("%s handler failed to send NotFound error to client: %v", r.URL.Path, err)
			}
			return
		}

		handler := handlers[index]
		r = r.WithContext(context.WithValue(r.Context(), paramMapType{}, param2value))
		handler(w, r)
	}
}

func newHTTPHandler(route Route, human bool, errorf func(format string, args ...interface{})) http.HandlerFunc {
	h := route.Handler
	t := route.Transport
	if t == nil {
		t = DefaultTransport
	}

	if m, ok := h.(*interfaceMethod); ok {
		h = m.Func()
	}

	handlerValue := reflect.ValueOf(h)
	handlerType := handlerValue.Type()
	validateHandler(handlerType, route.Path)

	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		req := reflect.New(handlerType.In(1).Elem()).Interface()
		ctx, err := t.DecodeRequest(ctx, r, req)
		if err != nil {
			errorf("%s %s handler failed to parse request: %v", r.Method, r.URL.Path, err)
			err = t.EncodeError(ctx, w, httpError{
				Code:    http.StatusBadRequest,
				Message: fmt.Sprintf("failed to parse request: %v", err),
			})
			if err != nil {
				errorf("%s %s handler failed to send parsing error to client: %v", r.Method, r.URL.Path, err)
			}
			return
		}

		results := handlerValue.Call([]reflect.Value{reflect.ValueOf(ctx), reflect.ValueOf(req)})
		resp := results[0].Interface()
		errReflect := results[1].Interface()

		if errReflect != nil {
			errorf("%s %s handler failed: %v", r.Method, r.URL.Path, errReflect)
			if err := t.EncodeError(ctx, w, errReflect.(error)); err != nil {
				errorf("%s %s handler failed to send handler error to client: %v", r.Method, r.URL.Path, err)
			}
			return
		}

		if err := t.EncodeResponse(ctx, w, resp); err != nil {
			errorf("%s %s handler failed to write response: %v", r.Method, r.URL.Path, err)
			return
		}
	}
}

type httpError struct {
	Code    int
	Message string
}

func (e httpError) HttpCode() int {
	return e.Code
}

func (e httpError) Error() string {
	return e.Message
}
