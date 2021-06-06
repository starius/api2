package main

import (
	"os"
	"path/filepath"

	"github.com/starius/api2"
	"github.com/starius/api2/example"
	"github.com/starius/api2/typegen"
)

func main() {
	api2.GenerateClient(example.GetRoutes)
	api2.GenerateTSClient(&api2.TypesGenConfig{
		OutDir:    "./ts-types",
		Blacklist: []api2.BlacklistItem{{Service: "Hello"}},
		Routes:    []interface{}{example.GetRoutes},
		Types: []interface{}{
			&example.CustomType{},
			&example.CustomType2{},
		},
		OnDone: func(options *api2.TypesGenConfig, parser *typegen.Parser, routes []api2.Route) {
			_ = os.RemoveAll(filepath.Join(options.OutDir, "schema.ts"))
			schemaFile, _ := os.OpenFile(filepath.Join(options.OutDir, "schema.ts"), os.O_WRONLY|os.O_CREATE, 0755)
			_ = typegen.PrintJDT(parser, schemaFile)
		},
	})
	api2.GenerateOpenApiSpec(&api2.TypesGenConfig{
		OutDir: "./openapi",
		Routes: []interface{}{example.GetRoutes},
		Types: []interface{}{
			&example.EchoRequest{},
			&example.CustomType2{},
		},
	})
}
