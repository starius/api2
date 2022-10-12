package api2

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"path"
	"reflect"

	"google.golang.org/protobuf/proto"
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

	// The transport used in this route. If Transport is not set, DefaultTransport
	// is used.
	Transport Transport

	// Meta is optional field to put arbitrary data about the route.
	// E.g. the list of users who are allowed to use the route.
	Meta map[string]interface{}
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

var (
	protoType      = reflect.TypeOf((*proto.Message)(nil)).Elem()
	readCloserType = reflect.TypeOf((*io.ReadCloser)(nil)).Elem()
)

func validateRequestResponse(structType reflect.Type, request bool) {
	var jsonFields, bodyFields []string
	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		hasJson := field.Tag.Get("json") != ""
		hasUseAsBody := field.Tag.Get("use_as_body") == "true"
		hasProtobuf := field.Tag.Get("is_protobuf") == "true"
		hasStream := field.Tag.Get("is_stream") == "true"
		hasQuery := field.Tag.Get("query") != ""
		hasHeader := field.Tag.Get("header") != ""
		hasCookie := field.Tag.Get("cookie") != ""

		if hasProtobuf && !hasUseAsBody {
			panic(fmt.Sprintf("field %s of struct %s: hasProtobuf=%v, so hasUseAsBody must also be %v", field.Name, structType.Name(), hasProtobuf, hasUseAsBody))
		}
		if hasProtobuf && !field.Type.ConvertibleTo(protoType) {
			panic(fmt.Sprintf("field %s of struct %s: hasProtobuf=%v, but its type %s is not convertible to proto.Message", field.Name, structType.Name(), hasProtobuf, field.Type))
		}

		if hasStream {
			if !hasUseAsBody {
				panic(fmt.Sprintf("field %s of struct %s: hasStream=%v, so hasUseAsBody must also be %v", field.Name, structType.Name(), hasStream, hasUseAsBody))
			}
			if !readCloserType.AssignableTo(field.Type) {
				panic(fmt.Sprintf("field %s of struct %s: hasStream=%v, but io.ReadCloser is not assignable to its type %s", field.Name, structType.Name(), hasStream, field.Type))
			}
		}

		if hasStream && hasProtobuf {
			panic(fmt.Sprintf("field %s of struct %s: hasProtobuf=%v and hasStream=%v, but they must not be used together", field.Name, structType.Name(), hasProtobuf, hasStream))
		}

		sum := 0
		for _, v := range []bool{hasJson, hasUseAsBody, hasQuery, hasHeader, hasCookie} {
			if v {
				sum++
			}
		}
		if sum > 1 {
			panic(fmt.Sprintf("field %s of struct %s: hasJson=%v, hasUseAsBody=%v, hasQuery=%v, hasHeader=%v, hasCookie=%v want at most one to be true", field.Name, structType.Name(), hasJson, hasUseAsBody, hasQuery, hasHeader, hasCookie))
		}
		if hasQuery && !request {
			panic(fmt.Sprintf("field %s of struct %s: hasQuery=%v, but query can only be used in requests", field.Name, structType.Name(), hasQuery))
		}
		if hasCookie && !request {
			panic(fmt.Sprintf("field %s of struct %s: hasCookie=%v, but cookie can only be used in requests", field.Name, structType.Name(), hasCookie))
		}
		if hasJson {
			jsonFields = append(jsonFields, field.Name)
		}
		if hasUseAsBody {
			bodyFields = append(bodyFields, field.Name)
		}
	}
	if len(bodyFields) > 1 {
		panic(fmt.Sprintf("struct %s has more than 1 use_as_body field: %v", structType.Name(), bodyFields))
	}
	if len(bodyFields) > 0 && len(jsonFields) > 0 {
		panic(fmt.Sprintf("struct %s has both json (%v) and use_as_body (%v) fields", structType.Name(), jsonFields, bodyFields))
	}
}

var DefaultTransport = &JsonTransport{}

type interfaceMethod struct {
	serviceValue reflect.Value
	methodName   string
}

func (m *interfaceMethod) Func() interface{} {
	if m.serviceValue.Kind() == reflect.Interface && m.serviceValue.IsNil() {
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
	if serviceType.Kind() == reflect.Ptr {
		serviceType = serviceType.Elem()
	}
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
	if m.serviceValue.Kind() == reflect.Struct {
		panic("pass a pointer to an interface or a pointer to a pointer to a struct")
	}
	_ = m.Func() // To panic asap.
	return &m
}
