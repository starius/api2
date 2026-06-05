package typegen

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	gots "github.com/starius/api2/typegen"
	"github.com/starius/api2/typegen/tests/types"

	spec "github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/require"
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

func TestTsIgnoredFieldType_absentFromTypeScript(t *testing.T) {
	p := gots.NewFromTypes(&types.WithTsIgnoredObject{})
	var buf bytes.Buffer
	gots.PrintTsTypes(p, &buf, func(t reflect.Type) string { return "" })
	output := buf.String()
	require.NotContains(t, output, "TsHiddenPayload")
	require.NotContains(t, output, "internal")
	require.Contains(t, output, "name")
}

func TestTsIgnoredEmbedded_absentFromTypeScript(t *testing.T) {
	p := gots.NewFromTypes(&types.WithTsIgnoredEmbedded{})
	var buf bytes.Buffer
	gots.PrintTsTypes(p, &buf, func(t reflect.Type) string { return "" })
	output := buf.String()
	require.NotContains(t, output, "TsHiddenEmbedded")
	require.NotContains(t, output, "secret")
	require.Contains(t, output, "name")
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
