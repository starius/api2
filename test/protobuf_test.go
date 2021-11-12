package api2

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/starius/api2"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestProtobuf(t *testing.T) {
	type ProtoRequest struct {
		Body *durationpb.Duration `use_as_body:"true" is_protobuf:"true"`
	}
	type ProtoResponse struct {
		Body *timestamppb.Timestamp `use_as_body:"true" is_protobuf:"true"`
	}

	t1 := time.Date(2020, time.July, 10, 11, 30, 0, 0, time.UTC)

	protoHandler := func(ctx context.Context, req *ProtoRequest) (res *ProtoResponse, err error) {
		// Add passed duration to t1 and pass result back.
		duration := req.Body.AsDuration()
		t2 := t1.Add(duration)
		return &ProtoResponse{
			Body: timestamppb.New(t2),
		}, nil
	}

	routes := []api2.Route{
		{Method: http.MethodPost, Path: "/proto", Handler: protoHandler},
	}

	mux := http.NewServeMux()
	api2.BindRoutes(mux, routes)
	server := httptest.NewServer(mux)
	defer server.Close()

	client := api2.NewClient(routes, server.URL)

	protoRes := &ProtoResponse{}
	err := client.Call(context.Background(), protoRes, &ProtoRequest{
		Body: durationpb.New(time.Second),
	})
	if err != nil {
		t.Errorf("request with protobuf failed: %v.", err)
	}
	got := protoRes.Body.AsTime().Format("2006-01-02 15:04:05")
	want := "2020-07-10 11:30:01"
	if got != want {
		t.Errorf("request with protobuf returned %q, want %q", got, want)
	}
}
