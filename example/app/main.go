package main

import (
	"log"
	"net/http"

	"github.com/starius/api2"
	"github.com/starius/api2/example"
)


func main() {
	service := example.NewEchoService(example.NewEchoRepository())
	routes := example.GetRoutes(service)
	api2.BindRoutes(http.DefaultServeMux, routes)
	log.Fatal(http.ListenAndServe(":8080", nil))
}