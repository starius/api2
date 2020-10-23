package api2

import (
	"context"
	"testing"
)

func TestGetMethod(t *testing.T) {
	type FooRequest struct {
	}
	type FooResponse struct {
	}
	var foo func(ctx context.Context, req *FooRequest) (*FooResponse, error)

	type BarArgs struct {
	}
	type BarReply struct {
	}
	var bar func(ctx context.Context, req *BarArgs) (*BarReply, error)

	type BazRequest struct {
	}
	type BazResponse struct {
	}
	type BazService interface {
		Baz(ctx context.Context, req *BazRequest) (*BazResponse, error)
	}
	var baz BazService

	var badFoo func(ctx context.Context, req *FooRequest) (*BarReply, error)
	var badFoo2 func(ctx context.Context, req []int) ([]string, error)

	cases := []struct {
		handler                 interface{}
		name, request, response string
		wantErr                 bool
	}{
		{
			handler:  foo,
			name:     "Foo",
			request:  "FooRequest",
			response: "FooResponse",
		},
		{
			handler:  bar,
			name:     "Bar",
			request:  "BarArgs",
			response: "BarReply",
		},
		{
			handler:  Method(&baz, "Baz"),
			name:     "Baz",
			request:  "BazRequest",
			response: "BazResponse",
		},
		{
			handler: badFoo,
			wantErr: true,
		},
		{
			handler: badFoo2,
			wantErr: true,
		},
	}

	for i, tc := range cases {
		name, request, response, err := getMethod(tc.handler)
		if tc.wantErr {
			if err == nil {
				t.Errorf("case %d: want error, not got", i)
			}
		} else if !tc.wantErr && err != nil {
			t.Errorf("case %d: got error: %v", i, err)
		} else if name != tc.name {
			t.Errorf("case %d: want name %q, got %q", i, tc.name, name)
		} else if request != tc.request {
			t.Errorf("case %d: want request %q, got %q", i, tc.request, request)
		} else if response != tc.response {
			t.Errorf("case %d: want response %q, got %q", i, tc.response, response)
		}
	}
}
