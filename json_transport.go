package api2

import (
	"bytes"
	"context"
	"encoding"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"reflect"
	"sync"
)

// JsonTransport implements interface Transport for JSON encoding of requests and responses.
//
// It recognizes GET parameter "human". If it is set, JSON in the response is
// pretty formatted.
//
// To redefine some methods, set corresponding fields in the struct:
//  &JsonTransport{RequestDecoder: func ...
type JsonTransport struct {
	RequestDecoder  func(context.Context, *http.Request, interface{}) (context.Context, error)
	ResponseEncoder func(context.Context, http.ResponseWriter, interface{}) error
	ErrorEncoder    func(context.Context, http.ResponseWriter, error) error
	RequestEncoder  func(ctx context.Context, method, url string, req interface{}) (*http.Request, error)
	ResponseDecoder func(context.Context, *http.Response, interface{}) error
	ErrorDecoder    func(context.Context, *http.Response) error
}

type humanType struct{}

func (h *JsonTransport) DecodeRequest(ctx context.Context, r *http.Request, req interface{}) (context.Context, error) {
	if h.RequestDecoder != nil {
		return h.RequestDecoder(ctx, r, req)
	}

	// Calling FormValue before parsing JSON "eats" r.Body if Content-Type is
	// application/x-www-form-urlencoded. This happens in curl for me.
	ctx = context.WithValue(ctx, humanType{}, r.FormValue("human") != "")

	if err := parseRequest(req, r.Body, r.URL.Query(), r.Header); err != nil {
		return ctx, err
	}

	return ctx, nil
}

func (h *JsonTransport) EncodeResponse(ctx context.Context, w http.ResponseWriter, res interface{}) error {
	if h.ResponseEncoder != nil {
		return h.ResponseEncoder(ctx, w, res)
	}

	forJson, err := writeQueryAndHeader(res, nil, w.Header())
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(w)
	if human := ctx.Value(humanType{}); human != nil && human.(bool) {
		encoder.SetIndent("", "  ")
	}
	return encoder.Encode(forJson)
}

type HttpError interface {
	HttpCode() int
}

func errorToCode(err error) int {
	var jsonErr *json.SyntaxError
	if errors.As(err, &jsonErr) {
		return http.StatusBadRequest
	}
	var httpErr HttpError
	if errors.As(err, &httpErr) {
		return httpErr.HttpCode()
	}
	return http.StatusInternalServerError
}

func (h *JsonTransport) EncodeError(ctx context.Context, w http.ResponseWriter, err error) error {
	if h.ErrorEncoder != nil {
		return h.ErrorEncoder(ctx, w, err)
	}

	code := errorToCode(err)
	return jsonError(w, code, "%v", err)
}

func (h *JsonTransport) EncodeRequest(ctx context.Context, method, urlStr string, req interface{}) (*http.Request, error) {
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
	forJson, err := writeQueryAndHeader(req, query, request.Header)
	if err != nil {
		return nil, err
	}
	request.URL.RawQuery = query.Encode()

	requestJSON, err := json.Marshal(forJson)
	if err != nil {
		return nil, err
	}
	body := bytes.NewReader(requestJSON)
	snapshot := *body
	request.ContentLength = int64(len(requestJSON))
	request.Body = ioutil.NopCloser(body)
	request.GetBody = func() (io.ReadCloser, error) {
		r := snapshot
		return ioutil.NopCloser(&r), nil
	}

	request.Header.Set("Content-Type", "application/json")
	return request, nil
}

func (h *JsonTransport) DecodeResponse(ctx context.Context, res *http.Response, response interface{}) error {
	if h.ResponseDecoder != nil {
		return h.ResponseDecoder(ctx, res, response)
	}

	if err := parseRequest(response, res.Body, nil, res.Header); err != nil {
		return err
	}

	return nil
}

func (h *JsonTransport) DecodeError(ctx context.Context, res *http.Response) error {
	if h.ErrorDecoder != nil {
		return h.ErrorDecoder(ctx, res)
	}

	buf, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}
	var msg errorMessage
	if err := json.Unmarshal(buf, &msg); err != nil {
		return fmt.Errorf("failed to decode error message %s, HTTP status %s: %v", string(buf), res.Status, err)
	}
	return fmt.Errorf("API returned error with HTTP status %s: %v", res.Status, msg.Error)
}

type errorMessage struct {
	Error string `json:"error"`
}

func jsonError(w http.ResponseWriter, code int, format string, args ...interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	errmsg := fmt.Sprintf(format, args...)
	return json.NewEncoder(w).Encode(errorMessage{errmsg})
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
	JsonMapping   []intMapping
	TypeForJson   reflect.Type
	BodyField     int
}

const noBodyField = -1

func prepare(objType reflect.Type) *preparedType {
	p := &preparedType{BodyField: noBodyField}
	jsonFields := make([]reflect.StructField, 0, objType.NumField())
	for i := 0; i < objType.NumField(); i++ {
		field := objType.Field(i)
		queryKey := field.Tag.Get("query")
		headerKey := field.Tag.Get("header")
		isBodyField := field.Tag.Get("use_as_body") == "true"
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
		} else if isBodyField {
			p.BodyField = i
		} else {
			// Add to JSON.
			p.JsonMapping = append(p.JsonMapping, intMapping{
				OrigField: i,
				JsonField: len(jsonFields),
			})
			jsonFields = append(jsonFields, field)
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
	} else {
		_, err := fmt.Sscanf(value, "%v", objPtr)
		return err
	}
}

func writeQueryAndHeader(objPtr interface{}, query url.Values, header http.Header) (interface{}, error) {
	objType := reflect.TypeOf(objPtr).Elem()
	p0, has := prepared.Load(objType)
	if !has {
		p0 = prepare(objType)
		prepared.Store(objType, p0)
	}
	p := p0.(*preparedType)

	objValue := reflect.ValueOf(objPtr).Elem()

	var jsonPtr interface{}
	if p.BodyField != noBodyField {
		// 'use_as_body' case.
		jsonPtr = objValue.Field(p.BodyField).Addr().Interface()
	} else if len(p.QueryMapping) == 0 && len(p.HeaderMapping) == 0 {
		// No query and header. Returning the original object.
		jsonPtr = objPtr
	} else {
		// JSON fields mixed with header and/or query fields.
		forJson := reflect.New(p.TypeForJson).Elem()
		for _, m := range p.JsonMapping {
			forJson.Field(m.JsonField).Set(objValue.Field(m.OrigField))
		}
		jsonPtr = forJson.Addr().Interface()
	}

	for _, m := range p.QueryMapping {
		value, err := toString(objValue.Field(m.Field).Interface())
		if err != nil {
			field := objType.Field(m.Field)
			return nil, fmt.Errorf("failed to marshal value for field %s: %w", field.Name, err)
		}
		query.Set(m.Key, value)
	}
	for _, m := range p.HeaderMapping {
		value, err := toString(objValue.Field(m.Field).Interface())
		if err != nil {
			field := objType.Field(m.Field)
			return nil, fmt.Errorf("failed to marshal value for field %s: %w", field.Name, err)
		}
		header.Set(m.Key, value)
	}

	return jsonPtr, nil
}

func parseRequest(objPtr interface{}, jsonReader io.Reader, query url.Values, header http.Header) error {
	objType := reflect.TypeOf(objPtr).Elem()
	p0, has := prepared.Load(objType)
	if !has {
		p0 = prepare(objType)
		prepared.Store(objType, p0)
	}
	p := p0.(*preparedType)

	objValue := reflect.ValueOf(objPtr).Elem()

	if p.BodyField != noBodyField {
		// 'use_as_body' case.
		jsonPtr := objValue.Field(p.BodyField).Addr().Interface()
		if err := json.NewDecoder(jsonReader).Decode(jsonPtr); err != nil {
			return err
		}
	} else if len(p.QueryMapping)+len(p.HeaderMapping) == objType.NumField() {
		// All the fields are query or header. No fields for JSON.
		// In this case JSON parsing is skipped.
	} else if len(p.QueryMapping) == 0 && len(p.HeaderMapping) == 0 {
		// No query and header. Parse JSON into the original structure.
		if err := json.NewDecoder(jsonReader).Decode(objPtr); err != nil {
			return err
		}
	} else {
		// JSON fields mixed with header and/or query fields.
		// Parse JSON into a temporary struct and copy fields into the original struct.
		jsonPtrValue := reflect.New(p.TypeForJson)
		if err := json.NewDecoder(jsonReader).Decode(jsonPtrValue.Interface()); err != nil {
			return err
		}
		jsonValue := jsonPtrValue.Elem()
		for _, m := range p.JsonMapping {
			objValue.Field(m.OrigField).Set(jsonValue.Field(m.JsonField))
		}
	}

	// Drain the reader in case we skipped parsing or something is left.
	io.Copy(ioutil.Discard, jsonReader)

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

	return nil
}
