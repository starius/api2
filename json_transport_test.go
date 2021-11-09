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
	type Anon struct {
		Foo string `json:"foo"`
	}

	type Person struct {
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		Age       int    `json:"age"`
	}

	cases := []struct {
		objPtr      interface{}
		query       bool
		request     bool
		wantJson    string
		replaceBody string
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
				Bar int  `query:"bar"`
				Baz bool `header:"baz"`
			}{
				Bar: 100,
				Baz: true,
			},
			query:       true,
			wantJson:    `{}`,
			replaceBody: " ",
			wantQuery: map[string][]string{
				"bar": []string{"100"},
			},
			wantHeader: map[string][]string{
				"Baz": []string{"true"},
			},
		},
		{
			objPtr:      &struct{}{},
			wantJson:    `{}`,
			replaceBody: " ",
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
				Foo CustomType `cookie:"foo"`
			}{
				Foo: CustomType(5),
			},
			request:  true,
			wantJson: `{}`,
			wantHeader: map[string][]string{
				"Cookie": []string{"foo=|||||"},
			},
		},

		{
			objPtr: &struct {
				Foo int    `cookie:"foo"`
				Bar string `header:"bar"`
			}{
				Foo: 5,
				Bar: "hi",
			},
			request:  true,
			wantJson: `{}`,
			wantHeader: map[string][]string{
				"Cookie": []string{"foo=5"},
				"Bar":    []string{"hi"},
			},
		},

		{
			objPtr: &struct {
				Foo int    `cookie:"foo"`
				Bar string `header:"bar"`
				Baz string `json:"baz"`
			}{
				Foo: 5,
				Bar: "hi",
				Baz: "gg",
			},
			request:  true,
			wantJson: `{"baz":"gg"}`,
			wantHeader: map[string][]string{
				"Cookie": []string{"foo=5"},
				"Bar":    []string{"hi"},
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

		{
			objPtr: &struct {
				Anon
			}{
				Anon: Anon{
					Foo: "aaa",
				},
			},
			wantJson: `{"foo":"aaa"}`,
		},

		{
			objPtr: &struct {
				Body []int `use_as_body:"true"`
			}{
				Body: []int{1, 2, 3},
			},
			wantJson: `[1,2,3]`,
		},
		{
			objPtr: &struct {
				Body map[string]int `use_as_body:"true"`
			}{
				Body: map[string]int{
					"key1": 100,
					"key2": 200,
				},
			},
			wantJson: `{"key1":100,"key2":200}`,
		},
		{
			objPtr: &struct {
				Body       map[string]int `use_as_body:"true"`
				ExtraField string
			}{
				Body: map[string]int{
					"key1": 100,
					"key2": 200,
				},
			},
			wantJson: `{"key1":100,"key2":200}`,
		},
		{
			objPtr: &struct {
				Body []int `use_as_body:"true"`
				Bar  int   `query:"bar"`
				Baz  bool  `header:"baz"`
			}{
				Body: []int{1, 2, 3},
				Bar:  500,
				Baz:  true,
			},
			wantJson: `[1,2,3]`,
			query:    true,
		},
		{
			objPtr: &struct {
				Body       []int `use_as_body:"true"`
				Bar        int   `query:"bar"`
				Baz        bool  `header:"baz"`
				ExtraField string
			}{
				Body: []int{1, 2, 3},
				Bar:  500,
				Baz:  true,
			},
			wantJson: `[1,2,3]`,
			query:    true,
		},
		{
			objPtr: &struct {
				Body []Person `use_as_body:"true"`
			}{
				Body: []Person{
					{FirstName: "Ivan", LastName: "Ivanov", Age: 55},
					{FirstName: "Petr", LastName: "Petrov", Age: 75},
				},
			},
			wantJson: `[{"first_name":"Ivan","last_name":"Ivanov","age":55},{"first_name":"Petr","last_name":"Petrov","age":75}]`,
		},
		{
			objPtr: &struct {
				Body []Person `use_as_body:"true"`
				Bar  int      `query:"bar"`
				Baz  bool     `header:"baz"`
			}{
				Body: []Person{
					{FirstName: "Ivan", LastName: "Ivanov", Age: 55},
					{FirstName: "Petr", LastName: "Petrov", Age: 75},
				},
				Bar: 500,
				Baz: true,
			},
			wantJson: `[{"first_name":"Ivan","last_name":"Ivanov","age":55},{"first_name":"Petr","last_name":"Petrov","age":75}]`,
			query:    true,
		},
		{
			objPtr: &struct {
				Body map[string]Person `use_as_body:"true"`
			}{
				Body: map[string]Person{
					"ivan.ivanov": {FirstName: "Ivan", LastName: "Ivanov", Age: 55},
				},
			},
			wantJson: `{"ivan.ivanov":{"first_name":"Ivan","last_name":"Ivanov","age":55}}`,
		},
		{
			objPtr: &struct {
				Body       map[string]Person `use_as_body:"true"`
				ExtraField string
			}{
				Body: map[string]Person{
					"ivan.ivanov": {FirstName: "Ivan", LastName: "Ivanov", Age: 55},
				},
			},
			wantJson: `{"ivan.ivanov":{"first_name":"Ivan","last_name":"Ivanov","age":55}}`,
		},
	}

	for i, tc := range cases {
		var query url.Values
		if tc.query {
			query = make(url.Values)
		}
		request, err := http.NewRequest("POST", "http://example.com", bytes.NewReader(nil))
		if err != nil {
			t.Fatalf("case %d: http.NewRequest failed: %v", i, err)
		}
		header := request.Header
		if !tc.request {
			request = nil
		}

		forJson, err := writeQueryHeaderCookie(tc.objPtr, query, request, header)
		if err != nil {
			t.Errorf("case %d: writeQueryHeaderCookie failed: %v", i, err)
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

		if tc.replaceBody != "" {
			jsonBytes = []byte(tc.replaceBody)
		}

		objPtr2 := reflect.New(reflect.TypeOf(tc.objPtr).Elem()).Interface()
		if err := parseRequest(objPtr2, bytes.NewReader(jsonBytes), query, request, header); err != nil {
			t.Errorf("case %d: parseRequest failed: %v", i, err)
		}

		if !tc.dontCompare && !reflect.DeepEqual(objPtr2, tc.objPtr) {
			t.Errorf("case %d: decoded object is not equal to source object: %#v != %#v", i, objPtr2, tc.objPtr)
		}
	}
}
