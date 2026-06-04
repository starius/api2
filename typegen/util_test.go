package typegen

import (
	"reflect"
	"testing"
)

func TestParseStructTag_tsIgnored(t *testing.T) {
	type testStruct struct {
		Hidden string `json:"hidden" ts:"-"`
	}
	field, _ := reflect.TypeOf(testStruct{}).FieldByName("Hidden")
	result, err := ParseStructTag(field.Tag)
	if err != nil {
		t.Fatal(err)
	}
	if result.State != TsIgnored {
		t.Errorf("expected TsIgnored, got %v", result.State)
	}
}

func TestParseStructTag_tsIgnoredWithHeader(t *testing.T) {
	type testStruct struct {
		RequestID string `header:"X-Request-Id" ts:"-"`
	}
	field, _ := reflect.TypeOf(testStruct{}).FieldByName("RequestID")
	result, err := ParseStructTag(field.Tag)
	if err != nil {
		t.Fatal(err)
	}
	if result.State != TsIgnored {
		t.Errorf("expected TsIgnored, got %v", result.State)
	}
	if result.FieldName != "X-Request-Id" {
		t.Errorf("expected FieldName X-Request-Id, got %q", result.FieldName)
	}
}

func TestParseStructTag_jsonIgnored(t *testing.T) {
	type testStruct struct {
		Hidden string `json:"-"`
	}
	field, _ := reflect.TypeOf(testStruct{}).FieldByName("Hidden")
	result, err := ParseStructTag(field.Tag)
	if err != nil {
		t.Fatal(err)
	}
	if result.State != Ignored {
		t.Errorf("expected Ignored, got %v", result.State)
	}
}
