package errors

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/starius/api2"
)

func TestErrToHttp(t *testing.T) {
	cases := []struct {
		err  error
		want int
	}{
		{
			err:  NotFound("document is not found"),
			want: http.StatusNotFound,
		},
		{
			err:  Internal("all shards failed"),
			want: http.StatusInternalServerError,
		},
		{
			err:  fmt.Errorf("can not find the document with ID 123: %w", NotFound("document is not found")),
			want: http.StatusNotFound,
		},

		// Other errors.
		{
			err:  io.EOF,
			want: http.StatusInternalServerError,
		},
		{
			err:  fmt.Errorf("some error"),
			want: http.StatusInternalServerError,
		},
		{
			err:  errors.New("some error"),
			want: http.StatusInternalServerError,
		},
	}

	transport := &api2.JsonTransport{}
	for _, tc := range cases {
		recorder := httptest.NewRecorder()
		if err := transport.EncodeError(context.Background(), recorder, tc.err); err != nil {
			t.Errorf("failed to encode err %v: %v", tc.err, err)
			continue
		}
		got := recorder.Result().StatusCode
		if got != tc.want {
			t.Errorf("for err %v got code %d, want %d", tc.err, got, tc.want)
		}
	}
}

func TestUnwrap(t *testing.T) {
	cases := []struct {
		err  error
		is   error
		want bool
	}{
		{
			err:  AlreadyExists("document already exists: %w", os.ErrExist),
			is:   os.ErrExist,
			want: true,
		},
		{
			err:  AlreadyExists("document already exists"),
			is:   os.ErrExist,
			want: false,
		},
	}

	for _, tc := range cases {
		got := errors.Is(tc.err, tc.is)
		if got != tc.want {
			t.Errorf("errors.Is(%v, %v) returned %v, want %v.", tc.err, tc.is, got, tc.want)
		}
	}
}
