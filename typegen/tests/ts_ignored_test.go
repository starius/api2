package typegen

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	gots "github.com/starius/api2/typegen"
	"github.com/starius/api2/typegen/tests/types"

	spec "github.com/getkin/kin-openapi/openapi3"
)

func TestTsIgnoredField_absentFromTypeScript(t *testing.T) {
	p := gots.NewFromTypes(&types.WithTsIgnoredField{})
	var buf bytes.Buffer
	gots.PrintTsTypes(p, &buf, func(t reflect.Type) string { return "" })
	output := buf.String()
	if strings.Contains(output, "internal") {
		t.Errorf("expected 'internal' field to be absent from TypeScript output, got:\n%s", output)
	}
	if !strings.Contains(output, "name") {
		t.Errorf("expected 'name' field to be present in TypeScript output, got:\n%s", output)
	}
}

func TestTsIgnoredComplexField_subtypeAbsentFromTypeScript(t *testing.T) {
	p := gots.NewFromTypes(&types.WithTsIgnoredComplexField{})
	var buf bytes.Buffer
	gots.PrintTsTypes(p, &buf, func(t reflect.Type) string { return "" })
	output := buf.String()
	if strings.Contains(output, "InternalDetail") {
		t.Errorf("expected 'InternalDetail' type to be absent from TypeScript output, got:\n%s", output)
	}
	if strings.Contains(output, "InternalStatus") {
		t.Errorf("expected 'InternalStatus' enum to be absent from TypeScript output, got:\n%s", output)
	}
	if !strings.Contains(output, "name") {
		t.Errorf("expected 'name' field to be present in TypeScript output, got:\n%s", output)
	}
}

func TestTsIgnoredComplexField_subtypePresentInSwagger(t *testing.T) {
	p := gots.NewFromTypes(&types.WithTsIgnoredComplexField{})
	swag := &spec.T{
		Components: &spec.Components{},
	}
	gots.PrintSwagger(p, swag)

	typeName := "types.WithTsIgnoredComplexField"
	schema, ok := swag.Components.Schemas[typeName]
	if !ok {
		t.Fatalf("schema %q not found in Swagger output", typeName)
	}
	if _, hasDetail := schema.Value.Properties["detail"]; !hasDetail {
		t.Errorf("expected 'detail' field to be present in Swagger schema, got properties: %v", schema.Value.Properties)
	}
	detailSchema, hasDetail := swag.Components.Schemas["types.InternalDetail"]
	if !hasDetail {
		t.Errorf("expected 'InternalDetail' type to be present in Swagger schemas")
	}
	if detailSchema != nil {
		if _, hasStatus := detailSchema.Value.Properties["status"]; !hasStatus {
			t.Errorf("expected 'status' field in InternalDetail Swagger schema")
		}
	}
}

func TestTsIgnoredField_presentInSwagger(t *testing.T) {
	p := gots.NewFromTypes(&types.WithTsIgnoredField{})
	swag := &spec.T{
		Components: &spec.Components{},
	}
	gots.PrintSwagger(p, swag)

	typeName := "types.WithTsIgnoredField"
	schema, ok := swag.Components.Schemas[typeName]
	if !ok {
		t.Fatalf("schema %q not found in Swagger output", typeName)
	}
	if _, hasInternal := schema.Value.Properties["internal"]; !hasInternal {
		t.Errorf("expected 'internal' field to be present in Swagger schema, got properties: %v", schema.Value.Properties)
	}
	if _, hasName := schema.Value.Properties["name"]; !hasName {
		t.Errorf("expected 'name' field to be present in Swagger schema")
	}
}
