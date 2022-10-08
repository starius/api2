package streamtransport

import (
	"bytes"
	"context"
	"encoding"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"reflect"
	"sync"
)

// &StreamTransport{RequestDecoder: func ...
type StreamTransport struct {
	RequestDecoder  func(context.Context, *http.Request, interface{}) (context.Context, error)
	ResponseEncoder func(context.Context, http.ResponseWriter, interface{}) error
	ErrorEncoder    func(context.Context, http.ResponseWriter, error) error
	RequestEncoder  func(ctx context.Context, method, url string, req interface{}) (*http.Request, error)
	ResponseDecoder func(context.Context, *http.Response, interface{}) error
	ErrorDecoder    func(context.Context, *http.Response) error

	// Errors whose structure is preserved and parsed back by api2 Client.
	// Values in the map are sample objects of error types. Keys in the map
	// are user-provided names of such errors. This value is passed in a
	// separate JSON field ("detail") as well as its type (in JSON field
	// "code"). Other errors are reduced to their messages.
	Errors map[string]error
}

type humanType struct{}

func (h *StreamTransport) DecodeRequest(ctx context.Context, r *http.Request, req interface{}) (context.Context, error) {
	if h.RequestDecoder != nil {
		return h.RequestDecoder(ctx, r, req)
	}

	// Calling FormValue before parsing JSON "eats" r.Body if Content-Type is
	// application/x-www-form-urlencoded. This happens in curl for me.
	ctx = context.WithValue(ctx, humanType{}, r.FormValue("human") != "")

	if err := parseRequest(req, r.Body, r.URL.Query(), r, r.Header); err != nil {
		return ctx, err
	}

	return ctx, nil
}

func (h *StreamTransport) EncodeResponse(ctx context.Context, w http.ResponseWriter, res interface{}) error {
	if h.ResponseEncoder != nil {
		return h.ResponseEncoder(ctx, w, res)
	}

	human := false
	if humanValue := ctx.Value(humanType{}); humanValue != nil {
		human = humanValue.(bool)
	}
	body, err := writeQueryHeaderCookie(res, nil, nil, w.Header(), human)
	if err != nil {
		return err
	}
	if body != nil {
		_, err = io.Copy(w, body)
	}

	return err
}

func (h *StreamTransport) EncodeRequest(ctx context.Context, method, urlStr string, req interface{}) (*http.Request, error) {
	if h.RequestEncoder != nil {
		return h.RequestEncoder(ctx, method, urlStr, req)
	}

	request, err := http.NewRequestWithContext(ctx, method, urlStr, nil)
	if err != nil {
		return nil, err
	}
	query, err := url.ParseQuery(request.URL.RawQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to parse query part of URL: %w", err)
	}

	body, err := writeQueryHeaderCookie(req, query, request, request.Header, false)
	if err != nil {
		return nil, err
	}
	request.URL.RawQuery = query.Encode()
	request.Body = body
	request.GetBody = func() (io.ReadCloser, error) {
		// todo check if it's correct
		b := bytes.Buffer{}
		io.Copy(&b, body)
		return ioutil.NopCloser(&b), nil
	}

	return request, nil
}

func (h *StreamTransport) DecodeResponse(ctx context.Context, res *http.Response, response interface{}) error {
	if h.ResponseDecoder != nil {
		return h.ResponseDecoder(ctx, res, response)
	}

	if err := parseRequest(response, res.Body, nil, nil, res.Header); err != nil {
		return err
	}

	return nil
}

type strMapping struct {
	Field int
	Key   string
}

type intMapping struct {
	OrigField int
	JsonField int
}

type preparedType struct {
	QueryMapping  []strMapping
	HeaderMapping []strMapping
	CookieMapping []strMapping
	JsonMapping   []intMapping
	TypeForJson   reflect.Type
	BodyField     int
	Protobuf      bool
}

const noBodyField = -1

func prepare(objType reflect.Type) *preparedType {
	p := &preparedType{BodyField: noBodyField}
	jsonFields := make([]reflect.StructField, 0, objType.NumField())
	for i := 0; i < objType.NumField(); i++ {
		field := objType.Field(i)
		if field.PkgPath != "" {
			// The field is unexported, according to https://golang.org/pkg/reflect/#StructField .
			continue
		}
		queryKey := field.Tag.Get("query")
		headerKey := field.Tag.Get("header")
		cookieKey := field.Tag.Get("cookie")
		isBodyField := field.Tag.Get("use_as_body") == "true"
		p.Protobuf = isBodyField && field.Tag.Get("is_protobuf") == "true"
		if queryKey != "" {
			p.QueryMapping = append(p.QueryMapping, strMapping{
				Field: i,
				Key:   queryKey,
			})
		} else if headerKey != "" {
			p.HeaderMapping = append(p.HeaderMapping, strMapping{
				Field: i,
				Key:   headerKey,
			})
		} else if cookieKey != "" {
			p.CookieMapping = append(p.CookieMapping, strMapping{
				Field: i,
				Key:   cookieKey,
			})
		} else if isBodyField {
			p.BodyField = i
		}
	}
	if p.BodyField == noBodyField {
		p.TypeForJson = reflect.StructOf(jsonFields)
	}
	return p
}

var prepared sync.Map

func toString(obj interface{}) (string, error) {
	if marshaler, ok := obj.(encoding.TextMarshaler); ok {
		valueBytes, err := marshaler.MarshalText()
		if err != nil {
			return "", err
		}
		return string(valueBytes), nil
	} else {
		return fmt.Sprintf("%v", obj), nil
	}
}

func fromString(objPtr interface{}, value string) error {
	if unmarshaler, ok := objPtr.(encoding.TextUnmarshaler); ok {
		return unmarshaler.UnmarshalText([]byte(value))
	} else if fieldStrPtr, ok := objPtr.(*string); ok {
		*fieldStrPtr = value
		return nil
	} else if value == "" {
		return nil
	} else {
		_, err := fmt.Sscanf(value, "%v", objPtr)
		return err
	}
}

func writeQueryHeaderCookie(objPtr interface{}, query url.Values, request *http.Request, header http.Header, human bool) (io.ReadCloser, error) {
	objType := reflect.TypeOf(objPtr).Elem()
	p0, has := prepared.Load(objType)
	if !has {
		p0 = prepare(objType)
		prepared.Store(objType, p0)
	}
	p := p0.(*preparedType)

	objValue := reflect.ValueOf(objPtr).Elem()
	var stream io.ReadCloser
	if p.BodyField != noBodyField {
		// 'use_as_body' case.
		fieldValue := objValue.Field(p.BodyField)
		if fieldValue.Kind() != reflect.Ptr {
			// Take a pointer if the field is not a pointer.
			// Protobuf does not parse into a double pointer,
			// that is why the check is needed.
			fieldValue = fieldValue.Addr()
		}
		bodyPtr := fieldValue.Interface()
		res, ok := bodyPtr.(*io.ReadCloser)
		if !ok {
			return nil, fmt.Errorf("use_as_body field must be of type io.ReadCloser")
		}
		stream = *res

	}

	err := writeQueryHeaderCookieInternal(p, objValue, objType, query, header, request)
	if err != nil {
		return nil, err
	}

	return stream, nil
}

func writeQueryHeaderCookieInternal(p *preparedType, objValue reflect.Value, objType reflect.Type, query url.Values, header http.Header, request *http.Request) error {
	for _, m := range p.QueryMapping {
		value, err := toString(objValue.Field(m.Field).Interface())
		if err != nil {
			field := objType.Field(m.Field)
			return fmt.Errorf("failed to marshal value for field %s: %w", field.Name, err)
		}
		query.Set(m.Key, value)
	}
	for _, m := range p.HeaderMapping {
		value, err := toString(objValue.Field(m.Field).Interface())
		if err != nil {
			field := objType.Field(m.Field)
			return fmt.Errorf("failed to marshal value for field %s: %w", field.Name, err)
		}
		header.Set(m.Key, value)
	}
	for _, m := range p.CookieMapping {
		value, err := toString(objValue.Field(m.Field).Interface())
		if err != nil {
			field := objType.Field(m.Field)
			return fmt.Errorf("failed to marshal value for field %s: %w", field.Name, err)
		}
		request.AddCookie(&http.Cookie{Name: m.Key, Value: value})
	}
	return nil
}

func parseRequest(objPtr interface{}, bodyReader io.ReadCloser, query url.Values, request *http.Request, header http.Header) error {
	objType := reflect.TypeOf(objPtr).Elem()
	p0, has := prepared.Load(objType)
	if !has {
		p0 = prepare(objType)
		prepared.Store(objType, p0)
	}
	p := p0.(*preparedType)
	var isStream bool
	objValue := reflect.ValueOf(objPtr).Elem()

	if p.BodyField != noBodyField {
		// 'use_as_body' case.
		fieldValue := objValue.Field(p.BodyField)
		if fieldValue.Kind() == reflect.Ptr {
			// Fill the pointer with new object.
			fieldValue.Set(reflect.New(fieldValue.Type().Elem()))
		} else {
			// Take a pointer if the field is not a pointer.
			// Protobuf does not parse into a double pointer,
			// that is why the check is needed.
			fieldValue = fieldValue.Addr()
		}
		bodyPtr := fieldValue.Interface()
		_, ok := bodyPtr.(*io.ReadCloser)
		isStream = ok
		if isStream {
			objValue.Field(p.BodyField).Set(reflect.ValueOf(bodyReader))
		}
	} else if len(p.QueryMapping)+len(p.HeaderMapping)+len(p.CookieMapping) == objType.NumField() {
		// All the fields are query, header or cookie. No fields for JSON.
		// In this case JSON parsing is skipped.
	} else if len(p.QueryMapping) == 0 && len(p.HeaderMapping) == 0 && len(p.CookieMapping) == 0 {
		// No query and header. Parse JSON into the original structure.
		if err := json.NewDecoder(bodyReader).Decode(objPtr); err != nil {
			return err
		}
	}

	err := parseQueryHeaderCookie(p, objValue, query, objType, header, request)
	if err != nil {
		return err
	}

	return nil
}

func parseQueryHeaderCookie(p *preparedType, objValue reflect.Value, query url.Values, objType reflect.Type, header http.Header, request *http.Request) error {
	for _, m := range p.QueryMapping {
		fieldPtr := objValue.Field(m.Field).Addr().Interface()
		value := query.Get(m.Key)
		if err := fromString(fieldPtr, value); err != nil {
			field := objType.Field(m.Field)
			return fmt.Errorf("failed to parse value %q from query key %s for field %s: %w", value, m.Key, field.Name, err)
		}
	}

	for _, m := range p.HeaderMapping {
		fieldPtr := objValue.Field(m.Field).Addr().Interface()
		value := header.Get(m.Key)
		if err := fromString(fieldPtr, value); err != nil {
			field := objType.Field(m.Field)
			return fmt.Errorf("failed to parse value %q from header key %s for field %s: %w", value, m.Key, field.Name, err)
		}
	}

	for _, m := range p.CookieMapping {
		fieldPtr := objValue.Field(m.Field).Addr().Interface()
		c, err := request.Cookie(m.Key)
		value := ""
		if err != http.ErrNoCookie {
			value = c.Value
		}
		if err := fromString(fieldPtr, value); err != nil {
			field := objType.Field(m.Field)
			return fmt.Errorf("failed to parse value %q from cookie key %s for field %s: %w", value, m.Key, field.Name, err)
		}
	}
	return nil
}
