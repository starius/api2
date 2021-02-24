package typegen

import (
	"bytes"
	"fmt"
	"io"
	"path"
	"reflect"
	"text/template"
)

const GlobalTemplate = `{{- range $namespace, $types := .}}
export declare namespace {{$namespace}} {
{{- range $type := $types}}
{{if $type.Doc| ne ""}}// {{$type.Doc -}}{{end}}
export type {{$type.Name}} = {{$type|Serialize}}{{end}}
}{{end}}
`

const RecordTemplate = `{{ range $field := .Embedded}} {{$field | RefName}} & {{end}}{
{{- range $field := .Fields}}
	{{$field | Row}}{{if $field.Doc| ne ""}} // {{$field.Doc}}{{end}}
{{- end}}
}
`

type TypeToString = func(t reflect.Type) string

func typeToString(t reflect.Type, getTypeName TypeToString, stringifyType TypeToString) string {
	k := t.Kind()
	customT := stringifyType(t)
	if customT != "" {
		return customT
	}
	switch {
	case k == reflect.Ptr:
		t = indirect(t)
		return fmt.Sprintf("%s | null", typeToString(t, getTypeName, stringifyType))
	case k == reflect.Struct:
		if isDate(t) {
			return "string"
		}
		return getTypeName(t)
	case isNumber(k) && isEnum(t):
		return getTypeName(t)
	case isNumber(k):
		return "number"
	case k == reflect.String && isEnum(t):
		return getTypeName(t)
	case k == reflect.String:
		return "string"
	case k == reflect.Bool:
		return "boolean"
	case k == reflect.Slice || k == reflect.Array:
		return fmt.Sprintf("Array<%s> | null", typeToString(t.Elem(), getTypeName, stringifyType))
	case k == reflect.Interface:
		return "any"
	case k == reflect.Map:
		KeyType, ValType := typeToString(t.Key(), getTypeName, stringifyType), typeToString(t.Elem(), getTypeName, stringifyType)
		return fmt.Sprintf("Record<%s, %s> |  null", KeyType, ValType)
	}
	return "any"
}

func stringifyCustom(t reflect.Type) string {
	return ""
}

type Stringifier = func(t reflect.Type) string

func PrintTsTypes(parser *Parser, w io.Writer, stringify Stringifier) {
	if stringify == nil {
		stringify = stringifyCustom
	}
	output := make(map[string][]IType)

	for _, m := range parser.visitOrder {
		pkg := parser.seen[m].GetPackage()
		output[path.Base(pkg)] = append(output[path.Base(pkg)], parser.seen[m])
	}

	recordToString := func(r *RecordDef) string {
		tmpl, err := template.New("content").Funcs(template.FuncMap{
			"RefName": func(t reflect.Type) string {
				return parser.GetVisited(t).RefName()
			},
			"Row": func(t RecordField) string {
				keyName := t.Key
				// fmt.Println("[printer]", t.Tag, t.Type, t)
				fieldType := t.Tag.FieldType
				if t.Tag.FieldName != "" {
					keyName = t.Tag.FieldName
				}
				// fmt.Println("[printer]", t.Tag, t.Type, t)

				if t.Type != nil {
					visited := parser.GetVisited(t.Type)
					if visited != nil {
						fieldType = visited.RefName()
					} else {
						// fmt.Println("[printer]", t.Tag, t.Type, visited)
						fieldType = typeToString(t.Type, func(t reflect.Type) string {
							// fmt.Println("[printer-tostring]", t, parser.GetVisited(t))
							return parser.GetVisited(t).RefName()
						}, stringify)
					}
				}
				optionalText := ""
				if t.Tag.State == Optional {
					optionalText = "?"
				}
				nullText := ""
				if t.Tag.State == Null || t.IsRef {
					nullText = " | null"
				}

				return fmt.Sprintf("%s%s: %s%s", keyName, optionalText, fieldType, nullText)
			},
		}).Parse(RecordTemplate)
		panicIf(err)
		w := &bytes.Buffer{}
		err = tmpl.Execute(w, r)
		panicIf(err)
		return w.String()
	}

	tmpl, err := template.New("types template").Funcs(template.FuncMap{
		"Serialize": func(t IType) string {
			switch v := t.(type) {
			case *RecordDef:
				return recordToString(v)
			case *EnumDef:
				res := ""
				for _, v := range v.Values {
					if res != "" {
						res += " | "
					}
					res += v.Stringify()
				}
				return res
			case *TypeDef:
				return typeToString(v.T, func(t reflect.Type) string {
					return parser.GetVisited(t).RefName()
				}, stringify)
			}
			return "1"
		},
	}).Parse(GlobalTemplate)
	panicIf(err)

	err = tmpl.Execute(w, output)
	panicIf(err)
}
