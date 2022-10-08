package example

//go:generate go run ./gen/...

import (
	"context"
	"io"
	"time"

	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
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

type SinceRequest struct {
	Session string                 `header:"session"`
	Body    *timestamppb.Timestamp `use_as_body:"true" is_protobuf:"true"`
}

type SinceResponse struct {
	Body *durationpb.Duration `use_as_body:"true" is_protobuf:"true"`
}

type StreamRequest struct {
	Key  string        `header:"authorization"`
	Body io.ReadCloser `use_as_body:"true"`
}

type StreamResponse struct {
}

type IEchoService interface {
	Hello(ctx context.Context, req *HelloRequest) (*HelloResponse, error)
	Echo(ctx context.Context, req *EchoRequest) (*EchoResponse, error)
	Since(ctx context.Context, req *SinceRequest) (*SinceResponse, error)
	StreamBody(ctx context.Context, req *StreamRequest) (*StreamResponse, error)
}
