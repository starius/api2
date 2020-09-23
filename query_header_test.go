package api2

import (
	"bytes"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"testing"
)

// CustomType is an integer encoded as a number of "|".
type CustomType int

func (c CustomType) MarshalText() (text []byte, err error) {
	return bytes.Repeat([]byte("|"), int(c)), nil
}

func (c *CustomType) UnmarshalText(text []byte) error {
	for _, b := range text {
		if b != '|' {
			return fmt.Errorf("found unknown characted: %v", b)
		}
	}
	*c = CustomType(len(text))
	return nil
}

func TestQueryAndHeader(t *testing.T) {

	cases := []struct {
		objPtr interface{}
		query  bool
	}{
		{
			objPtr: &struct {
				Foo string `query:"foo" json:"-"`
			}{
				Foo: "foo 12\n3",
			},
			query: true,
		},
		{
			objPtr: &struct {
				Foo string `header:"foo" json:"-"`
			}{
				Foo: "foo 123",
			},
		},

		{
			objPtr: &struct {
				Bar int  `query:"bar" json:"-"`
				Baz bool `header:"baz" json:"-"`
			}{
				Bar: 100,
				Baz: true,
			},
			query: true,
		},

		{
			objPtr: &struct {
				Foo int16 `query:"foo" json:"-"`
			}{
				Foo: -30,
			},
			query: true,
		},
		{
			objPtr: &struct {
				Foo bool `query:"foo" json:"-"`
			}{
				Foo: false,
			},
			query: true,
		},

		{
			objPtr: &struct {
				Foo CustomType `query:"foo" json:"-"`
			}{
				Foo: CustomType(5),
			},
			query: true,
		},
	}

	for i, tc := range cases {
		var query url.Values
		if tc.query {
			query = make(url.Values)
		}
		header := make(http.Header)

		if err := writeQueryAndHeader(tc.objPtr, query, header); err != nil {
			t.Errorf("case %d: writeQueryAndHeader failed: %v", i, err)
		}

		objPtr2 := reflect.New(reflect.TypeOf(tc.objPtr).Elem()).Interface()
		if err := parseQueryAndHeader(objPtr2, query, header); err != nil {
			t.Errorf("case %d: parseQueryAndHeader failed: %v", i, err)
		}

		if !reflect.DeepEqual(objPtr2, tc.objPtr) {
			t.Errorf("case %d: decoded object is not equal to source object: %#v != %#v", i, objPtr2, tc.objPtr)
		}
	}
}
