package typegen

import (
	"go/ast"
	"reflect"
	"regexp"
	"strings"
)

type Parser struct {
	rawTypes   []reflect.Type
	seen       map[reflect.Type]IType
	visitOrder []reflect.Type
	// You can skip field or replace it with another type
	CustomParse func(arg reflect.Type) (IType, bool)
}

func NewFromTypes(types ...interface{}) *Parser {
	p := &Parser{}
	p.seen = make(map[reflect.Type]IType)
	for _, rawType := range types {
		p.rawTypes = append(p.rawTypes, parseType(rawType))
	}

	p.Parse(p.rawTypes...)
	return p
}

func NewParser(types ...RawType) *Parser {
	p := &Parser{}
	p.seen = make(map[reflect.Type]IType)
	return p
}

func parseType(v interface{}) reflect.Type {
	var t reflect.Type
	switch v := v.(type) {
	case reflect.Type:
		t = v
	case reflect.Value:
		t = v.Type()
	default:
		t = indirect(reflect.TypeOf(v))
	}
	return t

}

func (this *Parser) ParseRaw(rawTypes ...interface{}) {
	for _, rawType := range rawTypes {
		t := parseType(rawType)
		this.visitType(t)
		this.rawTypes = append(this.rawTypes, t)
	}
}

func (this *Parser) Parse(rawTypes ...reflect.Type) {
	for _, rawType := range rawTypes {
		this.visitType(rawType)
		this.rawTypes = append(this.rawTypes, rawType)
	}
}

type Fn = func(t IType)

func (this *Parser) markVisit(t reflect.Type, v IType) {
	this.seen[t] = v
	this.visitOrder = append(this.visitOrder, t)
}

func (this *Parser) GetVisited(t reflect.Type) IType {
	return this.seen[t]
}

func (this *Parser) isVisited(t reflect.Type) bool {
	return this.seen[t] != nil
}

var re = regexp.MustCompile(`[\n\t\r]+`)

func FormatDoc(str string) string {
	doc := strings.TrimSpace(re.ReplaceAllString(str, " "))
	idoc := strings.ToLower(doc)
	if strings.HasPrefix(idoc, "deprecated") {
		// Turn leading "deprecated into @deprecated".
		doc = "@deprecated " + doc[len("deprecated"):]
	} else if strings.Contains(idoc, "deprecated") {
		doc = "@deprecated " + doc
	}
	return doc
}

func (this *Parser) visitType(t reflect.Type) {
	unrefT := indirect(t)
	k := unrefT.Kind()
	if this.isVisited(unrefT) {
		return
	}
	if this.CustomParse != nil {
		v, skip := this.CustomParse(unrefT)
		if !skip && v != nil {
			this.markVisit(unrefT, v)
			return
		} else if skip {
			return
		}
	}

	switch {
	case k == reflect.Struct:
		if isDate(unrefT) {
			break
		}
		record := &RecordDef{}
		record.Name = unrefT.Name()
		var astFields []*ast.Field
		// if we parse anonymous struct doc is not available
		if record.Name != "" {
			recordDoc, f := getFieldsAst(unrefT)
			if recordDoc != nil {
				astFields = f
				record.Doc = FormatDoc(recordDoc.Doc)
			}
		}
		record.T = unrefT
		this.markVisit(unrefT, record)
		for i := 0; i < unrefT.NumField(); i++ {
			var (
				structField     = unrefT.Field(i)
				structFieldType = indirect(structField.Type)
			)
			field := &RecordField{
				Key:   structField.Name,
				Type:  indirect(structFieldType),
				IsRef: structFieldType != structField.Type,
			}
			if record.Name != "" && astFields != nil && len(astFields) > i {
				field.Doc = FormatDoc(astFields[i].Comment.Text())
			}
			isEmbed := structField.Anonymous && k == reflect.Struct
			parseResult, err := ParseStructTag(structField.Tag)
			field.Tag = parseResult
			panicIf(err)
			if parseResult.State == Ignored || (parseResult.State == NoInfo && !isEmbed) {
				continue
			}
			if parseResult.FieldType != "" {
				// we should not parse field type if we set it manually
				field.Type = nil
				record.Fields = append(record.Fields, field)
				continue
			}
			this.visitType(structFieldType)
			// if struct type has no name it means it's anonymous so we set field value afterwards
			if structFieldType.Name() == "" && structFieldType.Kind() == reflect.Struct {
				this.GetVisited(structFieldType).SetName(record.Name+"_"+field.Key, unrefT.PkgPath())
			}
			if structField.Anonymous && k == reflect.Struct {
				record.Embedded = append(record.Embedded, structFieldType)
				continue
			}
			record.Fields = append(record.Fields, field)
		}
	case k == reflect.Map:
		this.visitType(unrefT.Key())
		fallthrough
	case k == reflect.Slice || k == reflect.Array:
		// shared with map
		this.visitType(unrefT.Elem())
		if unrefT.Name() != "" {
			b := &TypeDef{}
			b.Name = unrefT.Name()
			b.T = unrefT
			b.Doc = getDoc(unrefT).Doc
			this.markVisit(unrefT, b)
		}
	case (isNumber(k) || k == reflect.String) && isEnum(unrefT):
		{
			enum := &EnumDef{}
			this.markVisit(unrefT, enum)
			enum.T = unrefT
			if getDoc(unrefT) != nil {
				enum.Doc = getDoc(unrefT).Doc
			}
			enum.Values = getTypedEnumValues(t)
			enum.Name = unrefT.Name()
		}
	}

}
