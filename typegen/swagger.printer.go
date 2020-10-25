package typegen

import (
	"fmt"
	"reflect"

	"github.com/go-openapi/spec"
)

func PrintSwagger(p *Parser) spec.Swagger {
	def := spec.Definitions{}
	swag := spec.Swagger{SwaggerProps: spec.SwaggerProps{
		Definitions: def,
		Swagger:     "2.0",
	}}
	for _, st := range p.seen {
		schema := GenerateOpenApi(p, st)
		def[st.RefName()] = schema
	}
	return swag
}

func typeToSwagger(t reflect.Type, swaggerType spec.Schema, getTypeName TypeToString) spec.Schema {
	k := t.Kind()
	switch {
	case k == reflect.Ptr:
		t = indirect(t)
		swaggerType.AsNullable()
		return typeToSwagger(t, swaggerType, getTypeName)
	case k == reflect.Struct:
		if isDate(t) {
			swaggerType.AddType("string", "date-time")
			return swaggerType
		}
		ref, err := spec.NewRef("#/definitions/" + getTypeName(t))
		if err != nil {
			fmt.Println(err)
		}
		swaggerType.Ref = ref
		return swaggerType
	case isNumber(k) && isEnum(t):
		ref, err := spec.NewRef("#/definitions/" + getTypeName(t))
		if err != nil {
			fmt.Println(err)
		}
		swaggerType.Ref = ref
		return swaggerType
	case isNumber(k):
		return *swaggerType.Typed("number", "")
	case k == reflect.String && isEnum(t):
		ref, err := spec.NewRef("#/definitions/" + getTypeName(t))
		if err != nil {
			fmt.Println(err)
		}
		swaggerType.Ref = ref
		return swaggerType
	case k == reflect.String:
		return *swaggerType.Typed("string", "")
	case k == reflect.Bool:
		return *swaggerType.Typed("boolean", "")
	case k == reflect.Slice || k == reflect.Array:
		swaggerType.Type = spec.StringOrArray{"array"}
		props := spec.Schema{}
		swaggerType.Nullable = true
		swaggerType.CollectionOf(typeToSwagger(t.Elem(), props, getTypeName))
		return swaggerType
	case k == reflect.Map:
		props := spec.Schema{}
		item := typeToSwagger(t.Elem(), props, getTypeName)
		swaggerType.SchemaProps = spec.MapProperty(&item).SchemaProps
		return swaggerType
	}
	return swaggerType
}

func GenerateOpenApi(p *Parser, s IType) spec.Schema {
	t := spec.Schema{SchemaProps: spec.SchemaProps{
		Properties: map[string]spec.Schema{},
	}}
	propertiesTypes := &t
	switch v := s.(type) {
	case *EnumDef:
		t.Typed(v.T.Kind().String(), "")
		convertedValues := make([]interface{}, len(v.Values))
		for i, v := range v.Values {
			convertedValues[i] = v.value.Interface()
		}
		t.WithEnum(convertedValues...)
		return t
	case *RecordDef:
		if len(v.Embedded) != 0 {
			types := make([]spec.Schema, len(v.Embedded))
			for i, v := range v.Embedded {
				ref := spec.RefProperty("#/definitions/" + p.GetVisited(v).RefName())
				types[i] = *ref
			}
			propertiesTypes = &spec.Schema{SchemaProps: spec.SchemaProps{
				Properties: map[string]spec.Schema{},
			}}
			t.WithAllOf(types...)
			t.AddToAllOf(*propertiesTypes)
		}
		propertiesTypes.Typed("object", "")
		for _, field := range v.Fields {
			scm := spec.Schema{}
			propertiesTypes.SchemaProps.Properties[field.Key] = typeToSwagger(field.Type, scm, func(t reflect.Type) string {
				return p.GetVisited(t).RefName()
			})
		}
	}

	return t
}
