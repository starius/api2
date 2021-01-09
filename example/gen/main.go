package main

import (
	"github.com/starius/api2"
	"github.com/starius/api2/example"
)

func main() {
	api2.GenerateClient(example.GetRoutes)
	api2.GenerateTSClient(&api2.TsGenConfig{
		OutDir: "./ts-types",
		Routes: []interface{}{example.GetRoutes},
		Types: []interface{}{
			&example.CustomType{},
		},
	})
}
