package typegen

import (
	"fmt"
	"go/constant"
	"go/types"
	"path"
	"reflect"
	"strconv"
)

func getEnumValues(pkgName, typename string) ([]constant.Value, error) {
	res, err := GetPackages(pkgName)
	if err != nil {
		return nil, err
	}
	enums := []constant.Value{}
	pkg := res.Packages.Types.Scope()
	for _, name := range pkg.Names() {
		v := pkg.Lookup(name)
		// we could get even key name here to make more real world enums but it's fine as is
		baseTypename := path.Base(v.Type().String())
		if v != nil && baseTypename == typename {
			switch t := v.(type) {
			case *types.Const:
				{
					enums = append(enums, t.Val())
				}
			}
		}
	}
	return enums, nil
}

type EnumValue struct {
	value reflect.Value
}

func (this *EnumValue) Stringify() string {
	k := this.value.Kind()
	switch k {
	case reflect.String:
		return strconv.Quote(this.value.String())
	case reflect.Int:
		_, hasToString := this.value.Type().MethodByName("String")
		if hasToString {
			return strconv.Quote(fmt.Sprintf("%v", this.value))
		} else {
			return fmt.Sprintf("%d", this.value.Int())
		}
	}
	return ""
}

func getTypedEnumValues(t reflect.Type) []EnumValue {
	values, err := getEnumValues(t.PkgPath(), t.String())
	if err != nil {
		panic(err)
	}
	enumStrValues := []EnumValue{}
	for _, v := range values {
		reflectValue := reflect.New(t).Elem()
		switch t.Kind() {
		case reflect.String:
			reflectValue.SetString(constant.StringVal(v))
		case reflect.Int:
			value, ok := constant.Int64Val(v)
			if !ok {
				panic("failed to convert")
			}
			reflectValue.SetInt(value)
		default:
			// newVal := constant.Val(v)
			// fmt.Println(reflect.TypeOf(newVal), newVal, reflectValue, v.Kind(), t)
			panic("unknown type")
		}
		r := EnumValue{value: reflectValue}
		enumStrValues = append(enumStrValues, r)
	}
	return enumStrValues
}
