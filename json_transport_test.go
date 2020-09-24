package api2

import (
	"bytes"
	"encoding/json"
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
			return fmt.Errorf("found unknown character: %v", b)
		}
	}
	*c = CustomType(len(text))
	return nil
}

func TestQueryAndHeader(t *testing.T) {
	cases := []struct {
		objPtr      interface{}
		query       bool
		wantJson    string
		wantQuery   url.Values
		wantHeader  http.Header
		dontCompare bool // For cases of non-nil empty slice or map.
	}{
		{
			objPtr: &struct {
				Foo string `json:"foo"`
			}{
				Foo: "foo 123",
			},
			query:    true,
			wantJson: `{"foo":"foo 123"}`,
		},
		{
			objPtr: &struct {
				Foo string `query:"foo"`
			}{
				Foo: "foo 12\n3",
			},
			query: true,
			wantQuery: map[string][]string{
				"foo": []string{"foo 12\n3"},
			},
			wantJson: `{}`,
		},
		{
			objPtr: &struct {
				Foo string `header:"foo"`
			}{
				Foo: "foo 123",
			},
			wantHeader: map[string][]string{
				"Foo": []string{"foo 123"},
			},
			wantJson: `{}`,
		},

		{
			objPtr: &struct {
				Foo string `json:"foo"`
				Bar int    `query:"bar"`
				Baz bool   `header:"baz"`
			}{
				Foo: "foo!",
				Bar: 100,
				Baz: true,
			},
			query:    true,
			wantJson: `{"foo":"foo!"}`,
			wantQuery: map[string][]string{
				"bar": []string{"100"},
			},
			wantHeader: map[string][]string{
				"Baz": []string{"true"},
			},
		},
		{
			objPtr: &struct {
				Bar int  `query:"bar"`
				Baz bool `header:"baz"`
			}{
				Bar: 100,
				Baz: true,
			},
			query:    true,
			wantJson: `{}`,
			wantQuery: map[string][]string{
				"bar": []string{"100"},
			},
			wantHeader: map[string][]string{
				"Baz": []string{"true"},
			},
		},

		{
			objPtr: &struct {
				Foo int16 `query:"foo"`
			}{
				Foo: -30,
			},
			query:    true,
			wantJson: `{}`,
			wantQuery: map[string][]string{
				"foo": []string{"-30"},
			},
		},
		{
			objPtr: &struct {
				Foo bool `query:"foo"`
			}{
				Foo: false,
			},
			query:    true,
			wantJson: `{}`,
			wantQuery: map[string][]string{
				"foo": []string{"false"},
			},
		},

		{
			objPtr: &struct {
				Foo CustomType `query:"foo"`
			}{
				Foo: CustomType(5),
			},
			query:    true,
			wantJson: `{}`,
			wantQuery: map[string][]string{
				"foo": []string{"|||||"},
			},
		},

		{
			objPtr: &struct {
				Foo string `json:"foo"`
			}{
				Foo: "",
			},
			wantJson: `{"foo":""}`,
		},
		{
			objPtr: &struct {
				Foo string `json:"foo,omitempty"`
			}{
				Foo: "123",
			},
			wantJson: `{"foo":"123"}`,
		},
		{
			objPtr: &struct {
				Foo string `json:"foo,omitempty"`
			}{
				Foo: "",
			},
			wantJson: `{}`,
		},
		{
			objPtr: &struct {
				Foo int `json:"foo,omitempty"`
			}{
				Foo: 0,
			},
			wantJson: `{}`,
		},
		{
			objPtr: &struct {
				Foo bool `json:"foo,omitempty"`
			}{
				Foo: false,
			},
			wantJson: `{}`,
		},
		{
			objPtr: &struct {
				Foo []int `json:"foo,omitempty"`
			}{
				Foo: nil,
			},
			wantJson: `{}`,
		},
		{
			objPtr: &struct {
				Foo []int `json:"foo,omitempty"`
			}{
				Foo: []int{},
			},
			wantJson:    `{}`,
			dontCompare: true,
		},
		{
			objPtr: &struct {
				Foo map[string]int `json:"foo,omitempty"`
			}{
				Foo: nil,
			},
			wantJson: `{}`,
		},
		{
			objPtr: &struct {
				Foo map[string]int `json:"foo,omitempty"`
			}{
				Foo: map[string]int{},
			},
			wantJson:    `{}`,
			dontCompare: true,
		},
		{
			objPtr: &struct {
				Foo string `json:",omitempty"`
			}{
				Foo: "aaa",
			},
			wantJson: `{"Foo":"aaa"}`,
		},
		{
			objPtr: &struct {
				Foo string
			}{
				Foo: "aaa",
			},
			wantJson: `{"Foo":"aaa"}`,
		},
		{
			objPtr: &struct {
				Foo string `json:""`
			}{
				Foo: "aaa",
			},
			wantJson: `{"Foo":"aaa"}`,
		},
		{
			objPtr: &struct {
				Foo string `json:",omitempty"`
			}{
				Foo: "",
			},
			wantJson: `{}`,
		},
		{
			objPtr: &struct {
				Foo string `json:"-"`
			}{
				Foo: "",
			},
			wantJson: `{}`,
		},
		{
			objPtr: &struct {
				Foo string `json:"-,"`
			}{
				Foo: "ggg",
			},
			wantJson: `{"-":"ggg"}`,
		},
		{
			objPtr: &struct {
				Foo string `json:"-,"`
			}{
				Foo: "",
			},
			wantJson: `{"-":""}`,
		},
		{
			objPtr: &struct {
				Foo string `json:"-,omitempty"`
			}{
				Foo: "",
			},
			wantJson: `{}`,
		},
	}

	for i, tc := range cases {
		var query url.Values
		if tc.query {
			query = make(url.Values)
		}
		header := make(http.Header)

		forJson, err := writeQueryAndHeader(tc.objPtr, query, header)
		if err != nil {
			t.Errorf("case %d: writeQueryAndHeader failed: %v", i, err)
		}
		jsonBytes, err := json.Marshal(forJson)
		if err != nil {
			t.Errorf("case %d: json.Marshal failed: %v", i, err)
		}

		jsonStr := string(jsonBytes)
		if jsonStr != tc.wantJson {
			t.Errorf("case %d: got json %s, want %s", i, jsonStr, tc.wantJson)
		}
		if tc.query && tc.wantQuery != nil && !reflect.DeepEqual(query, tc.wantQuery) {
			t.Errorf("case %d: query does not match, got %#v, want %#v", i, query, tc.wantQuery)
		}
		if tc.wantHeader != nil && !reflect.DeepEqual(header, tc.wantHeader) {
			t.Errorf("case %d: header does not match, got %#v, want %#v", i, header, tc.wantHeader)
		}

		objPtr2 := reflect.New(reflect.TypeOf(tc.objPtr).Elem()).Interface()
		if err := json.Unmarshal(jsonBytes, objPtr2); err != nil {
			t.Errorf("case %d: json.Unmarshal failed: %v", i, err)
		}
		if err := parseQueryAndHeader(objPtr2, query, header); err != nil {
			t.Errorf("case %d: parseQueryAndHeader failed: %v", i, err)
		}

		if !tc.dontCompare && !reflect.DeepEqual(objPtr2, tc.objPtr) {
			t.Errorf("case %d: decoded object is not equal to source object: %#v != %#v", i, objPtr2, tc.objPtr)
		}
	}
}
