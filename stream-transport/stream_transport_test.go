package streamtransport

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"
)

// CustomType is an integer encoded as a number of "|".
type CustomType int

func (c CustomType) MarshalText() (text []byte, err error) {
	return bytes.Repeat([]byte("|"), int(c)), nil
}

type WithBody struct {
	Body io.ReadCloser
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

	// protobufSampleObject := timestamppb.New(time.Date(2020, time.July, 10, 11, 30, 0, 0, time.UTC))
	// protobufSampleBytes, err := proto.Marshal(protobufSampleObject)
	// if err != nil {
	// 	t.Errorf("failed to marshal protobufSampleObject: %v", err)
	// }

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
				Body io.ReadCloser `use_as_body:"true"`
				Auth string        `header:"Authorization"`
			}{
				Body: io.NopCloser(strings.NewReader("testing")),
				Auth: "Bearer 123",
			},
			query:    true,
			wantBody: string("testing"),
			// io reader goes into {}
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

		body, err := writeQueryHeaderCookie(tc.objPtr, query, request, header, false)
		if err != nil {
			t.Errorf("case %d: writeQueryHeaderCookie failed: %v", i, err)
		}
		bodyBytes, err := io.ReadAll(body)
		if err != nil {
			t.Errorf("case %d: io.ReadAll failed: %v", i, err)
		}
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
		if err := parseRequest(objPtr2, io.NopCloser(bytes.NewReader(bodyBytes)), query, request, header); err != nil {
			t.Errorf("case %d: parseRequest failed: %v", i, err)
		}

		bodyFromRequest := reflect.ValueOf(objPtr2).Elem().FieldByName("Body").Interface().(io.ReadCloser)
		var b bytes.Buffer
		_, err = io.Copy(&b, bodyFromRequest)
		if err != nil {
			t.Errorf("case %d: io.Copy failed: %v", i, err)
		}
		if b.String() != tc.wantBody {
			t.Errorf("case %d: got body %s, want %s", i, b.String(), tc.wantBody)
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
