package typegen

import (
	"encoding/json"
	"io"
	"reflect"
	"text/template"

	jdt "github.com/jsontypedef/json-typedef-go"
)

type JSONTypeDefSchema struct {
	Definitions          map[string]JSONTypeDefSchema `json:"definitions,omitempty"`
	Metadata             map[string]interface{}       `json:"metadata,omitempty"`
	Nullable             bool                         `json:"nullable,omitempty"`
	Ref                  *string                      `json:"ref,omitempty"`
	Type                 jdt.Type                     `json:"type,omitempty"`
	Enum                 []string                     `json:"enum,omitempty"`
	Elements             *JSONTypeDefSchema           `json:"elements,omitempty"`
	Properties           map[string]JSONTypeDefSchema `json:"properties,omitempty"`
	OptionalProperties   map[string]JSONTypeDefSchema `json:"optionalProperties,omitempty"`
	AdditionalProperties bool                         `json:"additionalProperties,omitempty"`
	Values               *JSONTypeDefSchema           `json:"values,omitempty"`
	Discriminator        string                       `json:"discriminator,omitempty"`
	Mapping              map[string]JSONTypeDefSchema `json:"mapping,omitempty"`
}

const GlobalSchemaTemplate = `// prettier-ignore 
import * as t from './gen'
import type {JTDSchema} from 'libs/validator'

export const schema =  {
{{range $name, $json := .Content}}{{$name}}: {{$json|Marshal}} as JTDSchema<t.{{$name|TypeName}}>,{{"\n"}}{{end}}}
`

func PrintJDT(p *Parser, writer io.Writer) error {
	def := &JSONTypeDefSchema{}
	def.Metadata = make(map[string]interface{})
	def.Definitions = make(map[string]JSONTypeDefSchema)
	for _, st := range p.seen {
		schema := genereateJDT(p, st)
		def.Definitions[st.IdName()] = *schema
	}

	tmpl, err := template.New("schema template").Funcs(template.FuncMap{
		"Marshal": func(v interface{}) string {
			a, _ := json.MarshalIndent(v, "", "  ")
			return string(a)
		},
		"TypeName": func(v string) string {
			for _, st := range p.seen {
				if st.IdName() == v {
					return st.RefName()
				}
			}
			return v
		},
	}).Parse(GlobalSchemaTemplate)
	panicIf(err)
	tmpl.Execute(writer, map[string]interface{}{
		"Content": def.Definitions,
	})
	return nil
}

func ToSchema(schema *JSONTypeDefSchema) (*jdt.Schema, error) {
	res, err := json.Marshal(schema)
	if err != nil {
		return nil, err
	}
	var jtdschema jdt.Schema
	err = json.Unmarshal(res, &jtdschema)
	if err != nil {
		return nil, err
	}
	return &jtdschema, nil
}

func typeToSchemaString(t reflect.Type, fieldSchema *JSONTypeDefSchema, getTypeName TypeToString) *JSONTypeDefSchema {
	k := t.Kind()

	if fieldSchema.Metadata == nil {
		fieldSchema.Metadata = make(map[string]interface{})
	}
	switch {
	case k == reflect.Ptr:
		t = indirect(t)
		fieldSchema.Nullable = true
		return fieldSchema
	case k == reflect.Struct:
		if isDate(t) {
			fieldSchema.Type = jdt.TypeTimestamp
			return fieldSchema
		}
		ref := getTypeName(t)
		fieldSchema.Ref = &ref
		return fieldSchema
	case isNumber(k) && isEnum(t):
		ref := getTypeName(t)
		fieldSchema.Ref = &ref
		return fieldSchema
	case isNumber(k):
		fieldSchema.Type = jdt.TypeFloat64
		return fieldSchema
	case k == reflect.String && isEnum(t):
		ref := getTypeName(t)
		fieldSchema.Ref = &ref
		return fieldSchema
	case k == reflect.String:
		fieldSchema.Type = jdt.TypeString
		return fieldSchema
	case k == reflect.Bool:
		fieldSchema.Type = jdt.TypeBoolean
		return fieldSchema
	case k == reflect.Slice || k == reflect.Array:
		fieldSchema.Elements = typeToSchemaString(t.Elem(), &JSONTypeDefSchema{}, getTypeName)
		return fieldSchema
	case k == reflect.Interface:
		return fieldSchema
	case k == reflect.Map:
		fieldSchema.Values = typeToSchemaString(t.Elem(), &JSONTypeDefSchema{}, getTypeName)
		return fieldSchema
	}
	return fieldSchema
}

func genereateJDT(p *Parser, s IType) *JSONTypeDefSchema {
	t := &JSONTypeDefSchema{}
	propertiesTypes := t
	t.Metadata = make(map[string]interface{})
	switch v := s.(type) {
	case *EnumDef:
		enumType := v.T.Kind().String()
		enumValues := []interface{}{}
		isInt := false
		for _, v := range v.Values {
			value, isIntValue := v.RawValue()
			if !isIntValue {
				val := value.(string)
				t.Enum = append(t.Enum, val)
			} else {
				isInt = true
				enumValues = append(enumValues, value)
			}
		}
		if isInt {
			t.Type = jdt.TypeInt32
		}
		t.Metadata["enumType"] = enumType
		if len(enumValues) > 0 {
			t.Metadata["enumValues"] = enumValues
		}
		return t
	case *RecordDef:
		if len(v.Embedded) != 0 {
			types := []string{}
			for _, v := range v.Embedded {
				types = append(types, p.GetVisited(v).IdName())
			}
			t.Metadata["allOf"] = types
			t.Properties = make(map[string]JSONTypeDefSchema)
		}

		propertiesTypes.Properties = make(map[string]JSONTypeDefSchema)
		for _, field := range v.Fields {
			scm := &JSONTypeDefSchema{}
			if field.Type == nil {
				continue
			}
			keyName := field.Key
			if field.Tag.FieldName != "" {
				keyName = field.Tag.FieldName
			}
			propertiesTypes.Properties[keyName] = *typeToSchemaString(field.Type, scm, func(t reflect.Type) string {
				return p.GetVisited(t).IdName()
			})
		}
	}
	return t
}
