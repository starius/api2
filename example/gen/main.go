package main

import (
	"github.com/starius/api2"
	"github.com/starius/api2/example"
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
