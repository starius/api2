package api2

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
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

	protobufSampleObject := timestamppb.New(time.Date(2020, time.July, 10, 11, 30, 0, 0, time.UTC))
	protobufSampleBytes, err := proto.Marshal(protobufSampleObject)
	if err != nil {
		t.Errorf("failed to marshal protobufSampleObject: %v", err)
	}

	cases := []struct {
		objPtr        interface{}
		query         bool
		request       bool
		wantBody      string
		replaceBody   string
		wantQuery     url.Values
		replaceQuery  url.Values
		wantHeader    http.Header
		replaceHeader http.Header
		cmpAsJson     bool // For cases of non-nil empty slice or map.
	}{
		{
			objPtr: &struct {
				Foo string `json:"foo"`
			}{
				Foo: "foo 123",
			},
			query:    true,
			wantBody: `{"foo":"foo 123"}`,
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
			wantBody: `{}`,
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
			wantBody: `{}`,
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
			wantBody: `{"foo":"foo!"}`,
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
			wantBody: `{}`,
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
			wantBody:    `{}`,
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
			wantBody:    `{}`,
			replaceBody: " ",
		},

		{
			objPtr: &struct {
				Foo int16 `query:"foo"`
			}{
				Foo: -30,
			},
			query:    true,
			wantBody: `{}`,
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
			wantBody: `{}`,
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
			wantBody: `{}`,
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
			wantBody: `{}`,
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
			wantBody: `{}`,
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
			wantBody: `{"baz":"gg"}`,
			wantHeader: map[string][]string{
				"Cookie": []string{"foo=5"},
				"Bar":    []string{"hi"},
			},
		},

		// Empty values.
		{
			objPtr: &struct {
				CookieInt    int    `cookie:"cookie_int"`
				CookieBool   bool   `cookie:"cookie_bool"`
				CookieString string `cookie:"cookie_string"`
				QueryInt     int    `query:"query_int"`
				QueryBool    bool   `query:"query_bool"`
				QueryString  string `query:"query_string"`
				HeaderInt    int    `header:"header_int"`
				HeaderBool   bool   `header:"header_bool"`
				HeaderString string `header:"header_string"`
			}{},
			request:       true,
			query:         true,
			wantBody:      `{}`,
			replaceHeader: map[string][]string{},
			replaceQuery:  map[string][]string{},
		},

		{
			objPtr: &struct {
				Foo string `json:"foo"`
			}{
				Foo: "",
			},
			wantBody: `{"foo":""}`,
		},
		{
			objPtr: &struct {
				Foo string `json:"foo,omitempty"`
			}{
				Foo: "123",
			},
			wantBody: `{"foo":"123"}`,
		},
		{
			objPtr: &struct {
				Foo string `json:"foo,omitempty"`
			}{
				Foo: "",
			},
			wantBody: `{}`,
		},
		{
			objPtr: &struct {
				Foo int `json:"foo,omitempty"`
			}{
				Foo: 0,
			},
			wantBody: `{}`,
		},
		{
			objPtr: &struct {
				Foo bool `json:"foo,omitempty"`
			}{
				Foo: false,
			},
			wantBody: `{}`,
		},
		{
			objPtr: &struct {
				Foo []int `json:"foo,omitempty"`
			}{
				Foo: nil,
			},
			wantBody: `{}`,
		},
		{
			objPtr: &struct {
				Foo []int `json:"foo,omitempty"`
			}{
				Foo: []int{},
			},
			wantBody:  `{}`,
			cmpAsJson: true,
		},
		{
			objPtr: &struct {
				Foo map[string]int `json:"foo,omitempty"`
			}{
				Foo: nil,
			},
			wantBody: `{}`,
		},
		{
			objPtr: &struct {
				Foo map[string]int `json:"foo,omitempty"`
			}{
				Foo: map[string]int{},
			},
			wantBody:  `{}`,
			cmpAsJson: true,
		},
		{
			objPtr: &struct {
				Foo string `json:",omitempty"`
			}{
				Foo: "aaa",
			},
			wantBody: `{"Foo":"aaa"}`,
		},
		{
			objPtr: &struct {
				Foo string
			}{
				Foo: "aaa",
			},
			wantBody: `{"Foo":"aaa"}`,
		},
		{
			objPtr: &struct {
				Foo string `json:""`
			}{
				Foo: "aaa",
			},
			wantBody: `{"Foo":"aaa"}`,
		},
		{
			objPtr: &struct {
				Foo string `json:",omitempty"`
			}{
				Foo: "",
			},
			wantBody: `{}`,
		},
		{
			objPtr: &struct {
				Foo string `json:"-"`
			}{
				Foo: "",
			},
			wantBody: `{}`,
		},
		{
			objPtr: &struct {
				Foo string `json:"-,"`
			}{
				Foo: "ggg",
			},
			wantBody: `{"-":"ggg"}`,
		},
		{
			objPtr: &struct {
				Foo string `json:"-,"`
			}{
				Foo: "",
			},
			wantBody: `{"-":""}`,
		},
		{
			objPtr: &struct {
				Foo string `json:"-,omitempty"`
			}{
				Foo: "",
			},
			wantBody: `{}`,
		},

		{
			objPtr: &struct {
				Anon
			}{
				Anon: Anon{
					Foo: "aaa",
				},
			},
			wantBody: `{"foo":"aaa"}`,
		},

		{
			objPtr: &struct {
				Body []int `use_as_body:"true"`
			}{
				Body: []int{1, 2, 3},
			},
			wantBody: `[1,2,3]`,
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
			wantBody: `{"key1":100,"key2":200}`,
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
			wantBody: `{"key1":100,"key2":200}`,
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
			wantBody: `[1,2,3]`,
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
			wantBody: `[1,2,3]`,
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
			wantBody: `[{"first_name":"Ivan","last_name":"Ivanov","age":55},{"first_name":"Petr","last_name":"Petrov","age":75}]`,
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
			wantBody: `[{"first_name":"Ivan","last_name":"Ivanov","age":55},{"first_name":"Petr","last_name":"Petrov","age":75}]`,
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
			wantBody: `{"ivan.ivanov":{"first_name":"Ivan","last_name":"Ivanov","age":55}}`,
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
			wantBody: `{"ivan.ivanov":{"first_name":"Ivan","last_name":"Ivanov","age":55}}`,
		},

		{
			objPtr: &struct {
				Foo *timestamppb.Timestamp `use_as_body:"true" is_protobuf:"true"`
			}{
				Foo: protobufSampleObject,
			},
			query:     true,
			wantBody:  string(protobufSampleBytes),
			cmpAsJson: true,
		},
		{
			objPtr: &struct {
				Foo *timestamppb.Timestamp `use_as_body:"true" is_protobuf:"true"`
				Bar string                 `header:"bar"`
			}{
				Foo: protobufSampleObject,
				Bar: "some field after is_protobuf field",
			},
			query:    true,
			wantBody: string(protobufSampleBytes),
			wantHeader: map[string][]string{
				"Bar": []string{"some field after is_protobuf field"},
			},
			cmpAsJson: true,
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

		var bodyBuffer bytes.Buffer
		if err := writeQueryHeaderCookie(&bodyBuffer, tc.objPtr, query, request, header, false); err != nil {
			t.Errorf("case %d: writeQueryHeaderCookie failed: %v", i, err)
		}
		bodyBytes := bytes.TrimSpace(bodyBuffer.Bytes())

		bodyStr := string(bodyBytes)
		if bodyStr != tc.wantBody {
			t.Errorf("case %d: got body %s (%v), want %s", i, bodyStr, bodyBytes, tc.wantBody)
		}
		if tc.query && tc.wantQuery != nil && !reflect.DeepEqual(query, tc.wantQuery) {
			t.Errorf("case %d: query does not match, got %#v, want %#v", i, query, tc.wantQuery)
		}
		if tc.wantHeader != nil {
			delete(header, "Accept")
			delete(header, "Content-Type")
			if !reflect.DeepEqual(header, tc.wantHeader) {
				t.Errorf("case %d: header does not match, got %#v, want %#v", i, header, tc.wantHeader)
			}
		}

		if tc.replaceBody != "" {
			bodyBytes = []byte(tc.replaceBody)
		}
		if tc.replaceHeader != nil {
			header = tc.replaceHeader
		}
		if tc.replaceQuery != nil {
			query = tc.replaceQuery
		}

		objPtr2 := reflect.New(reflect.TypeOf(tc.objPtr).Elem()).Interface()
		if err := parseRequest(objPtr2, bytes.NewReader(bodyBytes), query, request, header); err != nil {
			t.Errorf("case %d: parseRequest failed: %v", i, err)
		}

		gotJson, err := json.MarshalIndent(objPtr2, "", "  ")
		if err != nil {
			panic(err)
		}
		wantJson, err := json.MarshalIndent(tc.objPtr, "", "  ")
		if err != nil {
			panic(err)
		}

		var equal bool
		if tc.cmpAsJson {
			equal = bytes.Equal(gotJson, wantJson)
		} else {
			equal = reflect.DeepEqual(objPtr2, tc.objPtr)
		}

		if !equal {
			gotJson, err := json.MarshalIndent(objPtr2, "", "  ")
			if err != nil {
				panic(err)
			}
			wantJson, err := json.MarshalIndent(tc.objPtr, "", "  ")
			if err != nil {
				panic(err)
			}
			t.Errorf("case %d: decoded object is not equal to source object:\n got: %#v, %s\n want: %#v, %s", i, objPtr2, gotJson, tc.objPtr, wantJson)
		}
	}
}
