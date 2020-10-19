package main

import (
	"context"
	"fmt"

	"github.com/starius/api2/example"
)

func main() {
	client := example.NewClient("http://127.0.0.1:8080")

	echoRes, err := client.Echo(context.Background(), &example.EchoRequest{
		Text: "test",
	})
	if err != nil {
		panic(err)
	}

	fmt.Println(echoRes.Text)
}
