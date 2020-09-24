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
	"strings"
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

	if err := json.NewDecoder(r.Body).Decode(req); err != nil {
		return ctx, err
	}
	if err := parseQueryAndHeader(req, r.URL.Query(), r.Header); err != nil {
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

	if err := json.NewDecoder(res.Body).Decode(response); err != nil {
		return err
	}
	if err := parseQueryAndHeader(response, nil, res.Header); err != nil {
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

func writeQueryAndHeader(objPtr interface{}, query url.Values, header http.Header) (interface{}, error) {
	objType := reflect.TypeOf(objPtr).Elem()
	objValue := reflect.ValueOf(objPtr).Elem()
	forJson := make(map[string]interface{}, objType.NumField())
	for i := 0; i < objType.NumField(); i++ {
		field := objType.Field(i)

		headerKey := field.Tag.Get("header")
		queryKey := field.Tag.Get("query")
		if headerKey == "" && (query == nil || queryKey == "") {
			jsonTag := field.Tag.Get("json")
			if jsonTag == "-" {
				continue
			}
			parts := strings.SplitN(jsonTag, ",", 2)
			jsonKey := parts[0]
			if jsonKey == "" {
				jsonKey = field.Name
			}
			fieldValue := objValue.Field(i)
			if len(parts) == 2 && parts[1] == "omitempty" {
				if fieldValue.IsZero() {
					continue
				}
				kind := fieldValue.Kind()
				if kind == reflect.Array || kind == reflect.Map || kind == reflect.Slice || kind == reflect.String {
					if fieldValue.Len() == 0 {
						continue
					}
				}
			}
			forJson[jsonKey] = fieldValue.Interface()
			continue
		}

		fieldObj := objValue.Field(i).Interface()
		value := ""
		if marshaler, ok := fieldObj.(encoding.TextMarshaler); ok {
			valueBytes, err := marshaler.MarshalText()
			if err != nil {
				return nil, fmt.Errorf("failed to marshal value for field %s: %w", field.Name, err)
			}
			value = string(valueBytes)
		} else {
			value = fmt.Sprintf("%v", fieldObj)
		}

		if headerKey != "" {
			header.Set(headerKey, value)
		} else if query != nil && queryKey != "" {
			query.Set(queryKey, value)
		}
	}
	return forJson, nil
}

func parseQueryAndHeader(objPtr interface{}, query url.Values, header http.Header) error {
	objType := reflect.TypeOf(objPtr).Elem()
	objValue := reflect.ValueOf(objPtr).Elem()
	for i := 0; i < objType.NumField(); i++ {
		field := objType.Field(i)

		value := ""
		resetField := false
		if headerKey := field.Tag.Get("header"); headerKey != "" {
			value = header.Get(headerKey)
			resetField = true
		} else if query != nil {
			if queryKey := field.Tag.Get("query"); queryKey != "" {
				value = query.Get(queryKey)
				resetField = true
			}
		}
		if value == "" {
			if resetField {
				// Reset just in case it was provided in JSON.
				fieldValue := objValue.Field(i)
				fieldValue.Set(reflect.Zero(fieldValue.Type()))
			}
			continue
		}

		var err error
		fieldPtr := objValue.Field(i).Addr().Interface()
		if unmarshaler, ok := fieldPtr.(encoding.TextUnmarshaler); ok {
			err = unmarshaler.UnmarshalText([]byte(value))
		} else if fieldStrPtr, ok := fieldPtr.(*string); ok {
			*fieldStrPtr = value
		} else {
			_, err = fmt.Sscanf(value, "%v", fieldPtr)
		}
		if err != nil {
			return fmt.Errorf("failed to parse value %q for field %s: %w", value, field.Name, err)
		}
	}
	return nil
}
