package api2

import (
	"fmt"
	"net/http"
	"reflect"
)

// BindRoutes adds handlers of routes to http.ServeMux.
func BindRoutes(mux *http.ServeMux, routes []Route, opts ...Option) {
	config := NewDefaultConfig()
	for _, opt := range opts {
		opt(config)
	}
	errorf := config.errorf

	path2routes := make(map[string][]Route)
	for _, route := range routes {
		path2routes[route.Path] = append(path2routes[route.Path], route)
	}

	for path, routes := range path2routes {
		method2handler := make(map[string]http.HandlerFunc, len(routes))
		for _, route := range routes {
			if _, has := method2handler[route.Method]; has {
				panic(fmt.Sprintf("Duplicate pair (%s, %s)", path, route.Method))
			}
			method2handler[route.Method] = newHTTPHandler(route.Handler, route.Transport, errorf)
		}

		mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			handler, has := method2handler[r.Method]
			if !has {
				if err := jsonError(w, http.StatusMethodNotAllowed, "unsupported method: %v", r.Method); err != nil {
					errorf("%s handler failed to send MethodNotAllowed error to client: %v", r.URL.Path, err)
				}
				return
			}
			handler(w, r)
		})
	}
}

func newHTTPHandler(h interface{}, t Transport, errorf func(format string, args ...interface{})) http.HandlerFunc {
	if m, ok := h.(*interfaceMethod); ok {
		h = m.Func()
	}

	handlerValue := reflect.ValueOf(h)
	handlerType := handlerValue.Type()
	validateHandler(handlerType)

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
