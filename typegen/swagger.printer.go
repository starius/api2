package typegen

import (
	"reflect"

	spec "github.com/getkin/kin-openapi/openapi3"
)

const RefSchemaPrefix = "#/components/schemas/"
const RefReqPrefix = "#/components/requestBodies/"

func PrintSwagger(p *Parser, swag *spec.T) {
	def := spec.Schemas{}

	for _, st := range p.seen {
		schema := GenerateOpenApi(p, st)
		def[st.RefName()] = spec.NewSchemaRef("", &schema)
	}
	swag.Components.Schemas = def
}

func typeToSwagger(t reflect.Type, swaggerType *spec.SchemaRef, getTypeName TypeToString) *spec.SchemaRef {
	k := t.Kind()
	if swaggerType.Value == nil {
		swaggerType.Value = spec.NewSchema()
	}
	switch {
	case k == reflect.Ptr:
		t = indirect(t)
		swaggerType.Value.WithNullable()
		return typeToSwagger(t, swaggerType, getTypeName)
	case k == reflect.Struct:
		if isDate(t) {
			swaggerType.Value.Type = "string"
			swaggerType.Value.Format = "date-time"
			return swaggerType
		}

		swaggerType.Ref = RefSchemaPrefix + getTypeName(t)
		return swaggerType
	case isNumber(k) && isEnum(t):
		stringRef := RefSchemaPrefix + getTypeName(t)
		swaggerType.Ref = stringRef
		return swaggerType
	case isNumber(k):
		swaggerType.Value.Type = "number"
		return swaggerType
	case k == reflect.String && isEnum(t):
		swaggerType.Ref = RefSchemaPrefix + getTypeName(t)
		return swaggerType
	case k == reflect.String:
		swaggerType.Value.Type = "string"
		return swaggerType
	case k == reflect.Bool:
		swaggerType.Value.Type = "boolean"
		return swaggerType
	case k == reflect.Slice || k == reflect.Array:
		swaggerType.Value.Type = "array"
		props := &spec.SchemaRef{}
		swaggerType.Value.Items = typeToSwagger(t.Elem(), props, getTypeName)
		return swaggerType
	case k == reflect.Map:
		swaggerType.Value.AdditionalProperties = typeToSwagger(t.Elem(), &spec.SchemaRef{}, getTypeName)
		swaggerType.Value.Type = "object"
		return swaggerType
	}
	return swaggerType
}

func GenerateOpenApi(p *Parser, s IType) spec.Schema {
	t := spec.Schema{
		Properties: map[string]*spec.SchemaRef{},
	}
	propertiesTypes := &t
	switch v := s.(type) {
	case *EnumDef:
		convertedValues := make([]interface{}, len(v.Values))
		for i, v := range v.Values {
			convertedValues[i] = v.value.Interface()
		}
		enumType := v.T.Kind().String()
		if enumType != "string" {
			enumType = "number"
		}
		t.Type = enumType
		t.WithEnum(convertedValues...)
		return t
	case *RecordDef:
		if len(v.Embedded) != 0 {
			types := make([]*spec.SchemaRef, len(v.Embedded))
			for i, v := range v.Embedded {
				r := spec.NewSchemaRef(RefSchemaPrefix+p.GetVisited(v).RefName(), nil)
				types[i] = r
			}
			t.AllOf = append(t.OneOf, types...)
			t.Properties = map[string]*spec.SchemaRef{}
		}
		propertiesTypes.Type = "object"
		for _, field := range v.Fields {
			scm := spec.NewSchemaRef("", spec.NewSchema())
			if field.Type == nil {
				continue
			}
			keyName := field.Key
			if field.Tag.FieldName != "" {
				keyName = field.Tag.FieldName
			}
			propertiesTypes.Properties[keyName] = typeToSwagger(field.Type, scm, func(t reflect.Type) string {
				return p.GetVisited(t).RefName()
			})
			if propertiesTypes.Properties[keyName].Value != nil {
				propertiesTypes.Properties[keyName].Value.Description = field.Doc
			}
		}
	}
	return t
}
