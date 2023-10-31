package typegen

import (
	"encoding/json"
	"fmt"
	"go/constant"
	"go/types"
	"path"
	"reflect"
	"strconv"
	"strings"
)

type astEnum struct {
	name  string
	value constant.Value
}

func getEnumsFromAst(pkgName, typename string) ([]astEnum, error) {
	res, err := GetPackages(pkgName)
	if err != nil {
		return nil, err
	}
	pkg := res.Packages.Types.Scope()
	enums := make([]astEnum, 0, len(pkg.Names()))
	for _, name := range pkg.Names() {
		v := pkg.Lookup(name)
		// we could get even key name here to make more real world enums but it's fine as is
		baseTypename := path.Base(v.Type().String())
		if v != nil && baseTypename == typename {
			switch t := v.(type) {
			case *types.Const:
				{
					enums = append(enums, astEnum{
						name:  t.Name(),
						value: t.Val(),
					})
				}
			}
		}
	}

	// Drop the element if it is *Count - not a real member.
	enums2 := make([]astEnum, 0, len(enums))
	for _, e := range enums {
		if strings.HasSuffix(e.name, "Count") {
			continue
		}
		enums2 = append(enums2, e)
	}

	return enums2, nil
}

type EnumValue struct {
	name  string
	value reflect.Value
}

func (this *EnumValue) Stringify() string {
	switch this.value.Kind() {
	case reflect.String:
		return strconv.Quote(this.value.String())
	}

	switch t := this.value.Interface().(type) {
	case json.Marshaler:
		value, err := t.MarshalJSON()
		if err != nil {
			panic(err)
		}
		return string(value)
	case fmt.Stringer:
		return strconv.Quote(t.String())
	}

	switch this.value.Kind() {
	case reflect.Int, reflect.Int32:
		return fmt.Sprintf("%d", this.value.Int())
	}

	return ""
}

func getTypedEnumValues(t reflect.Type) []EnumValue {
	values, err := getEnumsFromAst(t.PkgPath(), t.String())
	if err != nil {
		panic(err)
	}
	enumStrValues := []EnumValue{}
	for _, v := range values {
		reflectValue := reflect.New(t).Elem()
		switch t.Kind() {
		case reflect.String:
			reflectValue.SetString(constant.StringVal(v.value))
		case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int:
			value, ok := constant.Int64Val(v.value)
			if !ok {
				panic("failed to convert")
			}
			reflectValue.SetInt(value)
		case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint:
			value, ok := constant.Uint64Val(v.value)
			if !ok {
				panic("failed to convert")
			}
			reflectValue.SetUint(value)
		default:
			// newVal := constant.Val(v)
			// fmt.Println(reflect.TypeOf(newVal), newVal, reflectValue, v.Kind(), t)
			panic("unknown type")
		}
		r := EnumValue{value: reflectValue, name: v.name}
		enumStrValues = append(enumStrValues, r)
	}
	return enumStrValues
}
