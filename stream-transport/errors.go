package streamtransport

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"reflect"

	"github.com/starius/api2/internal/shared"
)

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

func (h *StreamTransport) EncodeError(ctx context.Context, w http.ResponseWriter, err error) error {
	if h.ErrorEncoder != nil {
		return h.ErrorEncoder(ctx, w, err)
	}

	code := errorToCode(err)
	return h.jsonError(w, code, err)
}

func (h *StreamTransport) DecodeError(ctx context.Context, res *http.Response) error {
	if h.ErrorDecoder != nil {
		return h.ErrorDecoder(ctx, res)
	}

	buf, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}

	var msg shared.ErrorMessage
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

func (h *StreamTransport) jsonError(w http.ResponseWriter, code int, err error) error {
	unwrapped, errType := detectErrorType(err, h.Errors)

	msg := shared.ErrorMessage{Error: fmt.Sprintf("%v", err)}
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
