package example

//go:generate go run ./gen/...

import (
	"context"
	"time"
)

type Direction int

const (
	North Direction = iota
	East
	South
	West
)

type OpCode byte

const (
	Op_Read OpCode = iota + 1
	Op_Write
	Op_Add
)

type UserSettings map[string]interface{}
type CustomType struct {
	Hell int
	UserSettings
}

type CustomType2 struct {
	Hell int
	*UserSettings
}

type EchoRequest struct {
	Session  string               `header:"session"`
	Text     string               `json:"text"`
	internal string               //nolint:structcheck,unused
	Bar      time.Duration        `json:"bar"`
	Code     OpCode               `json:"code"`
	Dir      Direction            `json:"dir"`
	Items    []CustomType2        `json:"items"`
	Maps     map[string]Direction `json:"maps"`
}

// EchoResponse.
type EchoResponse struct {
	Text string `json:"text"` // field comment.
}

type HelloRequest struct {
	Key string `query:"key"`
}

type HelloResponse struct {
	Session string `header:"session"`
}

type IEchoService interface {
	Hello(ctx context.Context, req *HelloRequest) (*HelloResponse, error)
	Echo(ctx context.Context, req *EchoRequest) (*EchoResponse, error)
}
