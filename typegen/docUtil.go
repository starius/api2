package typegen

import (
	"go/ast"
	"go/doc"
	"reflect"
)

func getDoc(t reflect.Type) *doc.Type {
	res, err := GetPackages(t.PkgPath())
	panicIf(err)
	for _, docType := range res.Docs.Types {
		if docType.Name == t.Name() {
			return docType
		}
	}
	return nil
}

func getFieldsAst(t reflect.Type) (*doc.Type, []*ast.Field) {
	docType := getDoc(t)
	if docType == nil {
		return nil, nil
	}
	switch v := (docType.Decl.Specs[0]).(type) {
	case *ast.TypeSpec:
		{
			switch v2 := v.Type.(type) {
			case *ast.StructType:
				return docType, v2.Fields.List
			}
		}
	}
	return nil, nil
}
