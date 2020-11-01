package main

import (
	"context"
	"fmt"

	"github.com/starius/api2/example"
)

func main() {
	client, err := example.NewClient("http://127.0.0.1:8080")
	if err != nil {
		panic(err)
	}

	ctx := context.Background()

	helloRes, err := client.Hello(ctx, &example.HelloRequest{
		Key: "secret password",
	})

	echoRes, err := client.Echo(ctx, &example.EchoRequest{
		Session: helloRes.Session,
		Text:    "test",
	})
	if err != nil {
		panic(err)
	}

	fmt.Println(echoRes.Text)
}
