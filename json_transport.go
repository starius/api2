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
	"log"
	"net/http"
	"net/url"
	"reflect"
	"sync"

	"google.golang.org/protobuf/proto"
)

// JsonTransport implements interface Transport for JSON encoding of requests and responses.
//
// It recognizes GET parameter "human". If it is set, JSON in the response is
// pretty formatted.
//
// To redefine some methods, set corresponding fields in the struct:
//
//	&JsonTransport{RequestDecoder: func ...
type JsonTransport struct {
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

func (h *JsonTransport) DecodeRequest(ctx context.Context, r *http.Request, req interface{}) (context.Context, error) {
	if h.RequestDecoder != nil {
		return h.RequestDecoder(ctx, r, req)
	}

	// Calling FormValue before parsing JSON "eats" r.Body if Content-Type is
	// application/x-www-form-urlencoded. This happens in curl for me.
	ctx = context.WithValue(ctx, humanType{}, r.FormValue("human") != "")

	if err := readQueryHeaderCookie(req, r.Body, r.URL.Query(), r, r.Header, 0); err != nil {
		return ctx, err
	}

	return ctx, nil
}

func (h *JsonTransport) EncodeResponse(ctx context.Context, w http.ResponseWriter, res interface{}) error {
	if h.ResponseEncoder != nil {
		return h.ResponseEncoder(ctx, w, res)
	}

	human := false
	if humanValue := ctx.Value(humanType{}); humanValue != nil {
		human = humanValue.(bool)
	}
	body, err := writeQueryHeaderCookie(w, res, nil, nil, w.Header(), human)
	if body != nil {
		panic("unexpected body")
	}
	return err
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
	return h.jsonError(w, code, err)
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
	var requestBodyBuffer bytes.Buffer
	body, err := writeQueryHeaderCookie(&requestBodyBuffer, req, query, request, request.Header, false)
	if err != nil {
		return nil, err
	}
	request.URL.RawQuery = query.Encode()

	if body != nil {
		request.Body = body
	} else {
		requestBody := requestBodyBuffer.Bytes()
		body := bytes.NewReader(requestBody)
		snapshot := *body
		request.ContentLength = int64(len(requestBody))
		request.Body = ioutil.NopCloser(body)
		request.GetBody = func() (io.ReadCloser, error) {
			r := snapshot
			return ioutil.NopCloser(&r), nil
		}
	}

	return request, nil
}

func (h *JsonTransport) DecodeResponse(ctx context.Context, res *http.Response, response interface{}) error {
	if h.ResponseDecoder != nil {
		return h.ResponseDecoder(ctx, res, response)
	}

	if err := readQueryHeaderCookie(response, res.Body, nil, nil, res.Header, res.StatusCode); err != nil {
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

	errType := msg.Code
	errSample, has := h.Errors[errType]
	if has {
		errPtrValue := reflect.New(reflect.TypeOf(errSample))
		if err := json.Unmarshal(msg.Detail, errPtrValue.Interface()); err != nil {
			return fmt.Errorf("failed to decode error message %s of type %s: %v", string(msg.Detail), errType, err)
		}
		return errPtrValue.Elem().Interface().(error)
	} else {
		log.Printf("Unknown error type: %s", errType)
	}

	return fmt.Errorf("API returned error with HTTP status %s: %v", res.Status, msg.Error)
}

func detectErrorType(err error, registeredErrors map[string]error) (error, string) {
	for k, v := range registeredErrors {
		if reflect.TypeOf(v) == reflect.TypeOf(err) {
			return err, k
		}
	}
	err = errors.Unwrap(err)
	if err == nil {
		return nil, ""
	}
	return detectErrorType(err, registeredErrors)
}

func (h *JsonTransport) jsonError(w http.ResponseWriter, code int, err error) error {
	unwrapped, errType := detectErrorType(err, h.Errors)

	msg := errorMessage{Error: fmt.Sprintf("%v", err)}
	if errType != "" {
		origError, err2 := json.Marshal(unwrapped)
		if err2 != nil {
			log.Printf("Failed to serialize error of type %s: %v", errType, err2)
		} else {
			msg.Code = errType
			msg.Detail = origError
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	return json.NewEncoder(w).Encode(msg)
}

func (h *JsonTransport) BodyCloseNeeded(ctx context.Context, response, request interface{}) bool {
	objType := reflect.TypeOf(response).Elem()
	p0, has := prepared.Load(objType)
	if !has {
		p0 = prepare(objType)
		prepared.Store(objType, p0)
	}
	p := p0.(*preparedType)

	return !p.Stream
}

func (h *JsonTransport) DecodeResponseAndError(ctx context.Context, httpRes *http.Response, res interface{}) error {
	objType := reflect.TypeOf(res).Elem()
	p0, has := prepared.Load(objType)
	if !has {
		p0 = prepare(objType)
		prepared.Store(objType, p0)
	}
	p := p0.(*preparedType)

	if p.StatusField != noField {
		// If Response has status field, no HTTP statuses are considered errors.
		return h.DecodeResponse(ctx, httpRes, res)
	}

	if 200 <= httpRes.StatusCode && httpRes.StatusCode < 300 {
		// Handle all 2xx responses as success.
		return h.DecodeResponse(ctx, httpRes, res)
	} else {
		return h.DecodeError(ctx, httpRes)
	}
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
	StatusField   int
	Protobuf      bool
	Stream        bool

	// Special fields are query, header, cookie and status.
	NoJsonFields    bool
	NoSpecialFields bool
}

const noField = -1

func prepare(objType reflect.Type) *preparedType {
	p := &preparedType{BodyField: noField, StatusField: noField}
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
		isStatusField := field.Tag.Get("use_as_status") == "true"
		if isBodyField {
			p.Protobuf = field.Tag.Get("is_protobuf") == "true"
			p.Stream = field.Tag.Get("is_stream") == "true"
		}
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
		} else if isStatusField {
			p.StatusField = i
		} else {
			// Add to JSON.
			p.JsonMapping = append(p.JsonMapping, intMapping{
				OrigField: i,
				JsonField: len(jsonFields),
			})
			jsonFields = append(jsonFields, field)
		}
	}
	if p.BodyField == noField {
		p.TypeForJson = reflect.StructOf(jsonFields)
	}

	statusFields := 0
	if p.StatusField != noField {
		statusFields = 1
	}
	if len(p.QueryMapping)+len(p.HeaderMapping)+len(p.CookieMapping)+statusFields == objType.NumField() {
		p.NoJsonFields = true
	}
	if len(p.QueryMapping) == 0 && len(p.HeaderMapping) == 0 && len(p.CookieMapping) == 0 && p.StatusField == noField {
		p.NoSpecialFields = true
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

type headerWriter interface {
	WriteHeader(statusCode int)
}

func writeQueryHeaderCookie(w io.Writer, objPtr interface{}, query url.Values, request *http.Request, header http.Header, human bool) (io.ReadCloser, error) {
	header.Set("Content-Type", "application/json; charset=UTF-8")
	if request != nil {
		request.Header.Set("Accept", "application/json")
	}

	objType := reflect.TypeOf(objPtr).Elem()
	p0, has := prepared.Load(objType)
	if !has {
		p0 = prepare(objType)
		prepared.Store(objType, p0)
	}
	p := p0.(*preparedType)

	objValue := reflect.ValueOf(objPtr).Elem()

	var bodyPtr interface{}
	if p.BodyField != noField {
		// 'use_as_body' case.
		fieldValue := objValue.Field(p.BodyField)
		if !p.Stream && fieldValue.Kind() != reflect.Ptr {
			// Take a pointer if the field is not a pointer.
			// Protobuf does not parse into a double pointer,
			// that is why the check is needed.
			fieldValue = fieldValue.Addr()
		}
		bodyPtr = fieldValue.Interface()
	} else if p.NoSpecialFields {
		// Returning the original object.
		bodyPtr = objPtr
	} else {
		// JSON fields mixed with header and/or query fields.
		forJson := reflect.New(p.TypeForJson).Elem()
		for _, m := range p.JsonMapping {
			forJson.Field(m.JsonField).Set(objValue.Field(m.OrigField))
		}
		bodyPtr = forJson.Addr().Interface()
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
	for _, m := range p.CookieMapping {
		valueInterface := objValue.Field(m.Field).Interface()
		if request != nil {
			value, err := toString(valueInterface)
			if err != nil {
				field := objType.Field(m.Field)
				return nil, fmt.Errorf("failed to marshal value for field %s: %w", field.Name, err)
			}
			request.AddCookie(&http.Cookie{Name: m.Key, Value: value})
		} else {
			cookie := valueInterface.(http.Cookie)
			if cookie.Name == "" {
				cookie.Name = m.Key
			}
			if cookie.Name != m.Key {
				return nil, fmt.Errorf("wrong cookie name: want %q, got %q", m.Key, cookie.Name)
			}
			if v := cookie.String(); v != "" {
				header.Add("Set-Cookie", v)
			}
		}
	}

	if p.StatusField != noField {
		status := objValue.Field(p.StatusField).Interface().(int)
		if status == 0 {
			status = 200
		}
		w.(headerWriter).WriteHeader(status)
	}

	if p.Protobuf {
		bodyPtrMessage, ok := bodyPtr.(proto.Message)
		if !ok {
			panic("protobuf field is not of type proto.Message")
		}
		protoData, err := proto.Marshal(bodyPtrMessage)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal protobuf: %w", err)
		}
		_, err = w.Write(protoData)
		return nil, err
	} else if p.Stream {
		if bodyPtr == nil {
			bodyPtr = io.NopCloser(bytes.NewReader(nil))
		}
		bodyReadCloser := bodyPtr.(io.ReadCloser)
		if request != nil {
			// Client. Pass body reader to http.Request.Body.
			return bodyReadCloser, nil
		} else {
			// Server. Copy the body to w.
			_, err1 := io.Copy(w, bodyReadCloser)
			err2 := bodyReadCloser.Close()
			if err1 != nil && err2 != nil {
				return nil, fmt.Errorf("failed to write response body (error: %w) and close body reader (error: %v)", err1, err2)
			}
			if err1 != nil {
				return nil, fmt.Errorf("failed to write response body: %w", err1)
			}
			if err2 != nil {
				return nil, fmt.Errorf("failed to close response reader: %w", err2)
			}
			return nil, nil
		}
	} else {
		encoder := json.NewEncoder(w)
		if human {
			encoder.SetIndent("", "  ")
		}
		return nil, encoder.Encode(bodyPtr)
	}
}

func readQueryHeaderCookie(objPtr interface{}, bodyReadCloser io.ReadCloser, query url.Values, request *http.Request, header http.Header, status int) error {
	objType := reflect.TypeOf(objPtr).Elem()
	p0, has := prepared.Load(objType)
	if !has {
		p0 = prepare(objType)
		prepared.Store(objType, p0)
	}
	p := p0.(*preparedType)

	objValue := reflect.ValueOf(objPtr).Elem()

	if p.StatusField != noField {
		fieldValue := objValue.Field(p.StatusField)
		fieldValue.SetInt(int64(status))
	}
	if p.BodyField != noField {
		// 'use_as_body' case.
		fieldValue := objValue.Field(p.BodyField)
		if !p.Stream {
			if fieldValue.Kind() == reflect.Ptr {
				// Fill the pointer with new object.
				fieldValue.Set(reflect.New(fieldValue.Type().Elem()))
			} else {
				// Take a pointer if the field is not a pointer.
				// Protobuf does not parse into a double pointer,
				// that is why the check is needed.
				fieldValue = fieldValue.Addr()
			}
		}
		bodyPtr := fieldValue.Interface()
		if p.Protobuf {
			bodyPtrMessage, ok := bodyPtr.(proto.Message)
			if !ok {
				panic("protobuf field is not of type proto.Message")
			}
			buf, err := ioutil.ReadAll(bodyReadCloser)
			if err != nil {
				return err
			}
			if err := proto.Unmarshal(buf, bodyPtrMessage); err != nil {
				return err
			}
		} else if p.Stream {
			fieldValue.Set(reflect.ValueOf(bodyReadCloser))
		} else {
			if err := json.NewDecoder(bodyReadCloser).Decode(bodyPtr); err != nil {
				return err
			}
		}
	} else if p.NoJsonFields {
		// In this case JSON parsing is skipped.
	} else if p.NoSpecialFields {
		// Parse JSON into the original structure.
		if err := json.NewDecoder(bodyReadCloser).Decode(objPtr); err != nil {
			return err
		}
	} else {
		// JSON fields mixed with header and/or query fields.
		// Parse JSON into a temporary struct and copy fields into the original struct.
		jsonPtrValue := reflect.New(p.TypeForJson)
		if err := json.NewDecoder(bodyReadCloser).Decode(jsonPtrValue.Interface()); err != nil {
			return err
		}
		jsonValue := jsonPtrValue.Elem()
		for _, m := range p.JsonMapping {
			objValue.Field(m.OrigField).Set(jsonValue.Field(m.JsonField))
		}
	}

	if !p.Stream {
		// Drain the reader in case we skipped parsing or something is left.
		if _, err := io.Copy(ioutil.Discard, bodyReadCloser); err != nil {
			return err
		}
	}

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

	name2cookie := make(map[string]*http.Cookie)
	if request == nil && len(p.CookieMapping) != 0 {
		fakeResponse := http.Response{Header: header}
		cookies := fakeResponse.Cookies()
		for _, c := range cookies {
			name2cookie[c.Name] = c
		}
	}
	for _, m := range p.CookieMapping {
		fieldPtr := objValue.Field(m.Field).Addr().Interface()
		if request != nil {
			c, err := request.Cookie(m.Key)
			value := ""
			if err != http.ErrNoCookie {
				value = c.Value
			}
			if err := fromString(fieldPtr, value); err != nil {
				field := objType.Field(m.Field)
				return fmt.Errorf("failed to parse value %q from cookie key %s for field %s: %w", value, m.Key, field.Name, err)
			}
		} else {
			c, has := name2cookie[m.Key]
			if has {
				cookiePtr := fieldPtr.(*http.Cookie)
				*cookiePtr = *c
			}
		}
	}

	return nil
}
