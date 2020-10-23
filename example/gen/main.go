package main

import (
	"github.com/starius/api2"
	"github.com/starius/api2/example"
)

func main() {
	api2.GenerateClient(example.GetRoutes)
}
