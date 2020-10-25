package typegen

import (
	"go/doc"

	"golang.org/x/tools/go/packages"
)

type PkgParseResult struct {
	Packages *packages.Package
	Docs     *doc.Package
}

var cache = map[string]*PkgParseResult{}

func GetPackages(pkgName string) (*PkgParseResult, error) {
	if cache[pkgName] != nil {
		return cache[pkgName], nil
	}
	res, err := packages.Load(&packages.Config{
		Mode: packages.NeedTypes | packages.NeedSyntax,
	}, pkgName)

	if err != nil {
		return nil, err
	}
	docs, err := doc.NewFromFiles(res[0].Fset, res[0].Syntax, pkgName)
	if err != nil {
		return nil, err
	}
	cache[pkgName] = &PkgParseResult{
		Packages: res[0],
		Docs:     docs,
	}
	return cache[pkgName], nil
}
