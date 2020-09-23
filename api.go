package api2

import (
	"context"
	"fmt"
	"net/http"
	"path"
	"reflect"
)

// Route describes one endpoint in the API, associated with particular
// method of some service.
type Route struct {
	// HTTP method.
	Method string

	// HTTP path. The same path can be used multiple times with different methods.
	Path string

	// Handler is a function with the following signature:
	// func(ctx, *Request) (*Response, error)
	// Request and Response are custom structures, unique to this route.
	Handler interface{}

	// The transport used in this route.
	Transport Transport
}

// Transport converts back and forth between HTTP and Request, Response types.
type Transport interface {
	// Called by server.
	DecodeRequest(ctx context.Context, r *http.Request, req interface{}) (context.Context, error)
	EncodeResponse(ctx context.Context, w http.ResponseWriter, res interface{}) error
	EncodeError(ctx context.Context, w http.ResponseWriter, err error) error

	// Called by client.
	EncodeRequest(ctx context.Context, method, url string, req interface{}) (*http.Request, error)
	DecodeResponse(ctx context.Context, httpRes *http.Response, res interface{}) error
	DecodeError(ctx context.Context, httpRes *http.Response) error
}

var (
	contextType = reflect.TypeOf((*context.Context)(nil)).Elem()
	errorType   = reflect.TypeOf((*error)(nil)).Elem()
)

// validateHandler panics if handler is not of type func(ctx, *Request) (*Response, error)
func validateHandler(handlerType reflect.Type) {
	if handlerType.Kind() != reflect.Func {
		panic(fmt.Sprintf("handler is %s, want func", handlerType.Kind()))
	}

	if handlerType.NumIn() != 2 {
		panic(fmt.Sprintf("handler must have 2 arguments, got %d", handlerType.NumIn()))
	}
	if handlerType.In(0) != contextType {
		panic(fmt.Sprintf("handler's first argument must be context.Context, got %s", handlerType.Out(0)))
	}
	if handlerType.In(1).Elem().Kind() != reflect.Struct {
		panic(fmt.Sprintf("handler's second argument must be a pointer to a struct, got %s", handlerType.In(1)))
	}
	validateRequestResponse(handlerType.In(1).Elem(), true)

	if handlerType.NumOut() != 2 {
		panic(fmt.Sprintf("handler must have 2 results, got %d", handlerType.NumOut()))
	}
	if handlerType.Out(0).Elem().Kind() != reflect.Struct {
		panic(fmt.Sprintf("handler's first result must be a pointer to a struct, got %s", handlerType.Out(0)))
	}
	validateRequestResponse(handlerType.Out(0).Elem(), false)
	if handlerType.Out(1) != errorType {
		panic(fmt.Sprintf("handler's second argument must be error, got %s", handlerType.Out(1)))
	}
}

func validateRequestResponse(structType reflect.Type, request bool) {
	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		hasJson := field.Tag.Get("json") != ""
		hasQuery := field.Tag.Get("query") != ""
		hasHeader := field.Tag.Get("header") != ""
		sum := 0
		for _, v := range []bool{hasJson, hasQuery, hasHeader} {
			if v {
				sum++
			}
		}
		if sum > 1 {
			panic(fmt.Sprintf("field %s of struct %s: hasJson=%v, hasQuery=%v, hasHeader=%v, want at most one to be true", field.Name, structType.Name(), hasJson, hasQuery, hasHeader))
		}
		if hasQuery && !request {
			panic(fmt.Sprintf("field %s of struct %s: hasQuery=%v, but query can only be used in requests", field.Name, structType.Name(), hasQuery))
		}
	}
}

var DefaultTransport = &JsonTransport{}

type interfaceMethod struct {
	serviceValue reflect.Value
	methodName   string
}

func (m *interfaceMethod) Func() interface{} {
	if m.serviceValue.IsNil() {
		// Service is nil interface.
		serviceType := m.serviceValue.Type()
		method, has := serviceType.MethodByName(m.methodName)
		if !has {
			panic(fmt.Sprintf("Service type %s has no method %s", serviceType.Name(), m.methodName))
		}
		return reflect.New(method.Type).Elem().Interface()
	} else {
		// Service is a real type.
		return m.serviceValue.MethodByName(m.methodName).Interface()
	}
}

func (m *interfaceMethod) FuncInfo() (pkgFull, pkgName, structName, method string) {
	serviceType := m.serviceValue.Type()
	pkgFull = serviceType.PkgPath()
	pkgName = path.Base(pkgFull)
	structName = serviceType.Name()
	method = m.methodName
	return
}

func Method(servicePtr interface{}, methodName string) interface{} {
	m := interfaceMethod{
		serviceValue: reflect.ValueOf(servicePtr).Elem(),
		methodName:   methodName,
	}
	_ = m.Func() // To panic asap.
	return &m
}
