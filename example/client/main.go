package main

import (
	"context"
	"fmt"

	"github.com/starius/api2"
	"github.com/starius/api2/example"
)

func main() {
	routes := example.GetRoutes(nil)
	client := api2.NewClient(routes, "http://127.0.0.1:8080")

	echoRes := &example.EchoResponse{}
	err := client.Call(context.Background(), echoRes, &example.EchoRequest{
		Text: "test",
	})
	if err != nil {
		panic(err)
	}

	fmt.Println(echoRes.Text)
}
