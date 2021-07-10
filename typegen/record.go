package typegen

import (
	"path"
	"reflect"
	"strings"
)

type RawType struct {
	T reflect.Type
}

type IType interface {
	IsType()
	GetName() string
	SetName(name, pkg string)
	GetPackage() string
	RefName() string
	IdName() string
	GetType() reflect.Type
}

type BaseType struct {
	Doc     string
	Package string
	Name    string
	T       reflect.Type // indirect type
}

type TypeDef struct {
	BaseType
}

type RecordDef struct {
	BaseType
	Fields   []*RecordField
	Embedded []reflect.Type
}

type EnumDef struct {
	BaseType
	Values []EnumValue
}

type Package struct {
	Name  string
	Types []IType
}

type RecordField struct {
	Doc   string
	Key   string
	Tag   *ParseResult
	Type  reflect.Type
	IsRef bool
}

func (*RecordField) IsType() {}
func (*RecordDef) IsType()   {}
func (*EnumDef) IsType()     {}
func (*BaseType) IsType()    {}

func (this *BaseType) GetName() string {
	if this.Name == "" {
		return this.T.Name()
	}
	return this.Name
}

func (this *BaseType) SetName(s, pkg string) {
	this.Name = s
	this.Package = pkg
}

func (this *BaseType) RefName() string {
	pkg := this.GetPackage()
	return path.Base(pkg) + "." + this.Name
}

func (this *BaseType) IdName() string {
	pkg := this.GetPackage()
	return strings.Title(path.Base(pkg)) + this.Name
}

func (this *BaseType) GetType() reflect.Type {
	return this.T
}

func (this *BaseType) GetPackage() string {
	pkg := this.T.PkgPath()
	if pkg == "" {
		pkg = this.Package
	}
	return pkg
}
