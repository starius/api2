package example

//go:generate go run ./gen/...

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Direction int

// Values of Direction.
//
// Enum members can be skipped in tx and openapi.json:
// typegen_ignore(East)
// typegen_ignore(West)
const (
	North Direction = iota
	East
	South
	West
	DirectionCount
)

type OpCode byte

const (
	Op_Read OpCode = iota + 1
	Op_Write
	Op_Add
)

func (o OpCode) String() string {
	if o == Op_Read {
		return "read"
	} else if o == Op_Write {
		return "write"
	} else if o == Op_Add {
		return "add"
	} else {
		return ""
	}
}

type Color byte

// typegen_ignore(ColorGreen)
const (
	ColorRed Color = iota + 1
	ColorGreen
	ColorBlue
)

// If a type has both MarshalJSON and String, MarshalJSON is used
// to produce enum values.

func (c Color) String() string {
	if c == ColorRed {
		return "red"
	} else if c == ColorGreen {
		return "green"
	} else if c == ColorBlue {
		return "blue"
	} else {
		return ""
	}
}

func (c Color) MarshalJSON() ([]byte, error) {
	if c == ColorRed {
		return []byte(`"color_red"`), nil
	} else if c == ColorGreen {
		return []byte(`"color_green"`), nil
	} else if c == ColorBlue {
		return []byte(`"color_blue"`), nil
	} else {
		return []byte(`""`), nil
	}
}

func (c *Color) UnmarshalJSON(data []byte) error {
	if bytes.Equal(data, []byte(`"color_red"`)) {
		*c = ColorRed
	} else if bytes.Equal(data, []byte(`"color_green"`)) {
		*c = ColorGreen
	} else if bytes.Equal(data, []byte(`"color_blue"`)) {
		*c = ColorBlue
	} else if bytes.Equal(data, []byte(`""`)) {
		*c = 0
	} else {
		return fmt.Errorf("unknown color")
	}
	return nil
}

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
	User     string               `url:"user"`
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
	Text  string `json:"text"` // field comment.
	Old   string `json:"old"`  // Deprecated! Use field Text.
	Old2  string `json:"old2"` // The field is DEPRECATED!
	Color Color  `json:"color"`
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
	Session string        `header:"session"`
	Body    io.ReadCloser `use_as_body:"true" is_stream:"true"`
}

type StreamResponse struct {
	Body io.ReadCloser `use_as_body:"true" is_stream:"true"`
}

type RedirectRequest struct {
	ID string `query:"id"`
}

type RedirectResponse struct {
	Status int    `use_as_status:"true"`
	URL    string `header:"Location"`
}

type RawRequest struct {
	Token []byte `use_as_body:"true" is_raw:"true"`
}

type RawResponse struct {
	Token []byte `use_as_body:"true" is_raw:"true"`
}

type IEchoService interface {
	Hello(ctx context.Context, req *HelloRequest) (*HelloResponse, error)
	Echo(ctx context.Context, req *EchoRequest) (*EchoResponse, error)
	Since(ctx context.Context, req *SinceRequest) (*SinceResponse, error)
	Stream(ctx context.Context, req *StreamRequest) (*StreamResponse, error)
	Redirect(ctx context.Context, req *RedirectRequest) (*RedirectResponse, error)
	Raw(ctx context.Context, req *RawRequest) (*RawResponse, error)
}
