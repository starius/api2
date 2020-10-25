package api2

import (
	"path"
	"reflect"
	"runtime"
	"strings"
)

type FnInfo struct {
	PkgFull    string
	PkgName    string
	StructName string
	Method     string
}

type FuncInfoer interface {
	FuncInfo() (pkgFull, pkgName, structName, method string)
}

func GetFnInfo(i interface{}) FnInfo {
	if f, ok := i.(FuncInfoer); ok {
		pkgFull, pkgName, structName, method := f.FuncInfo()
		return FnInfo{
			PkgFull:    pkgFull,
			PkgName:    pkgName,
			StructName: structName,
			Method:     method,
		}
	}

	// Format: pgkpath.(object).Method-fm.
	funcName := runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
	lastSlash := strings.LastIndexByte(funcName, '/')
	if lastSlash < 0 {
		lastSlash = 0
	}
	lastDot := strings.LastIndexByte(funcName[lastSlash:], '.') + lastSlash
	baseNameWithService := path.Base(funcName[:lastDot])
	lastDotInService := strings.LastIndexByte(baseNameWithService, '.')
	lastMinusInName := strings.LastIndexByte(funcName, '-')
	pkgName := baseNameWithService[:lastDotInService]
	replacer := strings.NewReplacer("(", "", ")", "", "*", "")
	serviceName := replacer.Replace(baseNameWithService[lastDotInService+1:])
	pkgBase := pkgName
	method := funcName[lastDot+1 : lastMinusInName]
	pkgFull := funcName[:lastDot]
	return FnInfo{
		PkgFull:    pkgFull,
		PkgName:    pkgBase,
		StructName: serviceName,
		Method:     method,
	}
}
