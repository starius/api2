package api2

import (
	"reflect"
	"testing"
)

func TestValidateRequestResponse(t *testing.T) {
	cases := []struct {
		obj       interface{}
		request   bool
		wantPanic bool
	}{
		{
			obj: struct {
				Foo string `json:"foo"`
			}{},
			request: true,
		},
		{
			obj: struct {
				Foo string `json:"foo"`
			}{},
			request: false,
		},
		{
			obj: struct {
				Foo string `query:"foo"`
			}{},
			request: true,
		},
		{
			obj: struct {
				Foo string `query:"foo"`
			}{},
			request:   false,
			wantPanic: true,
		},
		{
			obj: struct {
				Foo string `header:"foo"`
			}{},
			request: true,
		},
		{
			obj: struct {
				Foo string `header:"foo"`
			}{},
			request: false,
		},
		{
			obj: struct {
				Foo string `json:"foo" query:"foo"`
			}{},
			request:   true,
			wantPanic: true,
		},
		{
			obj: struct {
				Foo string `header:"foo" query:"foo"`
			}{},
			request:   true,
			wantPanic: true,
		},
		{
			obj: struct {
				Foo string `json:"foo" header:"foo"`
			}{},
			request:   true,
			wantPanic: true,
		},
		{
			obj: struct {
				Foo string `json:"foo" header:"foo" query:"foo"`
			}{},
			request:   true,
			wantPanic: true,
		},
	}

	for i, tc := range cases {
		var message interface{}
		gotPanic := func() (gotPanic bool) {
			defer func() {
				if r := recover(); r != nil {
					gotPanic = true
					message = r
				}
			}()
			validateRequestResponse(reflect.TypeOf(tc.obj), tc.request)
			return
		}()
		if gotPanic != tc.wantPanic {
			t.Errorf("case %d: gotPanic=%v, wantPanic=%v, message=%v", i, gotPanic, tc.wantPanic, message)
		}
	}
}
