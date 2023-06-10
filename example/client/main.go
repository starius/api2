package main

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/starius/api2/example"
	"google.golang.org/protobuf/types/known/timestamppb"
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
	if err != nil {
		panic(err)
	}

	_, err = client.Echo(ctx, &example.EchoRequest{
		Session: helloRes.Session,
		Text:    "test",
	})
	if err == nil {
		panic("expected an error")
	}

	echoRes, err := client.Echo(ctx, &example.EchoRequest{
		Session: helloRes.Session,
		User:    "good-user",
		Text:    "test",
	})
	if err != nil {
		panic(err)
	}
	fmt.Println(echoRes.Text)

	sinceRes, err := client.Since(ctx, &example.SinceRequest{
		Session: helloRes.Session,
		Body:    timestamppb.New(time.Date(2020, time.July, 10, 11, 30, 0, 0, time.UTC)),
	})
	if err != nil {
		panic(err)
	}
	fmt.Println(sinceRes.Body.AsDuration())

	streamRes, err := client.Stream(ctx, &example.StreamRequest{
		Session: helloRes.Session,
		Body:    io.NopCloser(strings.NewReader("abc xyz")),
	})
	if err != nil {
		panic(err)
	}
	streamResBytes, err := io.ReadAll(streamRes.Body)
	if err != nil {
		panic(err)
	}
	if err := streamRes.Body.Close(); err != nil {
		panic(err)
	}
	fmt.Println(string(streamResBytes))

	redirectRes, err := client.Redirect(ctx, &example.RedirectRequest{
		ID: "user123",
	})
	if err != nil {
		panic(err)
	}
	fmt.Println(redirectRes.Status, redirectRes.URL)
}
