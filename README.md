# api2
[![godoc](https://godoc.org/https://github.com/starius/api2?status.svg)](https://godoc.org/github.com/starius/api2)

Go library to make HTTP API clients and servers.

Package api2 provides types and functions used to define interfaces of
client-server API and facilitate creation of server and client for it.
You define a common structure (GetRoutes, see below) in Go and `api2` makes
both HTTP client and HTTP server for you. You do not have to do JSON
encoding-decoding yourself and to duplicate schema information (data types
and path) in client.

## How to use this package.

Organize your code in services. Each service provides
some domain specific functionality. It is a Go type whose methods correspond
to exposed RPC's of the API. Each method has the following signature:

```go
func(ctx, *Request) (*Response, error)
```

Let's define a service Foo with method Bar.

```go
type Foo struct {
	...
}

type BarRequest struct {
	// These fields are stored in JSON format in body.
	Name string `json:"name"`

	// These fields are GET parameters.
	UserID int `query:"user_id"`

	// These fields are headers.
	FileHash string `header:"file_hash"`

	// These fields are cookies.
	Foo string `cookie:"foo"`

	// URL parameters present in URL template like "/path/:product".
	Product string `url:"product"`

	// These fields are skipped.
	SkippedField int `json:"-"`
}

type BarResponse struct {
	// These fields are stored in JSON format in body.
	FileSize int `json:"file_size"`

	// These fields are headers.
	FileHash string `header:"file_hash"`

	// These fields are skipped.
	SkippedField int `json:"-"`
}

func (s *Foo) Bar(ctx context.Context, req *BarRequest) (*BarResponse, error) {
	...
}
```

A field must not have more than one of tags: `json`, `query`, `header`, `cookie`.
Fields in query, header and cookie parts are encoded and decoded with
`fmt.Sprintf` and `fmt.Sscanf`. Strings are not decoded with `fmt.Sscanf`,
but passed as is. Types implementing `encoding.TextMarshaler` and
`encoding.TextUnmarshaler` are encoded and decoded using it.
Cookie in Response part must be of type `http.Cookie`.
If no field is no JSON field in the struct, then HTTP body is skipped.

You can also set HTTP status code of response by adding a field of type
`int` with tag `use_as_status:"true"` to Response. 0 is interpreted as 200.
If Response has status field, no HTTP statuses are considered errors.

If you need the top-level type matching body JSON to be not a struct,
but of some other kind (e.g. slice or map), you should provide a field
in your struct with tag `use_as_body:"true"`:

```go
type FooRequest struct {
	// Body of the request is JSON array of strings: ["abc", "eee", ...].
	Body []string `use_as_body:"true"`

	// You can add 'header', 'query' and 'cookie' fields here, but not 'json'.
}
```

If you use `use_as_body:"true"`, you can also set `is_protobuf:"true"`
and put a protobuf type (convertible to proto.Message) in that field.
It will be sent over wire as protobuf binary form.

You can add `use_as_body:"true" is_raw:"true"` to a `[]byte` field,
then it will keep the whole HTTP body.

**Streaming**. If you use `use_as_body:"true"`, you can also set
`is_stream:"true"`. In this case the field must be of type `io.ReadCloser`.
On the client side put any object implementing `io.ReadCloser` to such
a field in Request. It will be read and closed by the library and used
as HTTP request body. On the server side your handler
should read from the reader passed in that field of Request.
(You don't have to read the entire body and to close it.)
For Response, on the server side, the handler must put any object
implementing `io.ReadCloser` to such a field of Response.
The library will use it to generate HTTP response's body and close it.
On the client side your code must read from that reader the entire response
and then close it. If a streaming field is left `nil`, it is interpreted
as empty body.

Now let's write the function that generates the table of routes:

```go
func GetRoutes(s *Foo) []api2.Route {
	return []api2.Route{
		{
			Method:    http.MethodPost,
			Path:      "/v1/foo/bar/:product",
			Handler:   s.Bar,
			Transport: &api2.JsonTransport{},
		},
	}
}
```

You can add multiple routes with the same path, but in this case their
HTTP methods must be different so that they can be distinguished.

If `Transport` is not set, `DefaultTransport` is used which is defined as
`&api2.JsonTransport{}`.

**Error handling**. A handler can return any Go error. `JsonTransport`
by default returns JSON. `Error()` value is put into "error" field of
that JSON. If the error has `HttpCode() int` method, it is called and
the result is used as HTTP return code.
You can pass error details (any struct). For that the error must be of a
custom type. You should register the error type in `JsonTransport.Errors`
map. The key used for that error is put into "code" key of JSON and the
object of the registered type - into "detail" field. The error can be
wrapped using `fmt.Errorf("%w" ...)`. See
[custom_error_test.go](test/custom_error_test.go) for an example.

In the server you need a real instance of service Foo to pass to GetRoutes.
Then just bind the routes to http.ServeMux and run the server:

```go
// Server.
foo := NewFoo(...)
routes := GetRoutes(foo)
api2.BindRoutes(http.DefaultServeMux, routes)
log.Fatal(http.ListenAndServe(":8080", nil))
```

The server is running.
It serves foo.Bar function on path /v1/foo/bar with HTTP method Post.

Now let's create the client:

```go
// Client.
routes := GetRoutes(nil)
client := api2.NewClient(routes, "http://127.0.0.1:8080")
barRes := &BarResponse{}
err := client.Call(context.Background(), barRes, &BarRequest{
	Product: "product1",
	...
})
if err != nil {
	panic(err)
}
// Server's response is in variable barRes.
```

The client sent request to path "/v1/foo/bar/product1", from which
the server understood that product=product1.

Note that you don't have to pass a real service object to GetRoutes
on client side. You can pass nil, it is sufficient to pass all needed
information about request and response types in the routes table, that
is used by client to find a proper route.

You can make `GetRoutes` accepting an interface instead of a concrete
Service type. In this case you can not get method handlers by `s.Bar`,
because this code `panic`s if s is nil interface. As a workaround `api2`
provides function `Method(service pointer, methodName)` which you can use:

```go
type Service interface {
	Bar(ctx context.Context, req *BarRequest) (*BarResponse, error)
}

func GetRoutes(s Service) []api2.Route {
	return []api2.Route{
		{Method: http.MethodPost, Path: "/v1/foo/bar/:product", Handler: api2.Method(&s, "Bar"), Transport: &api2.JsonTransport{}},
	}
}
```

If you have function `GetRoutes` in package `foo` as above you can generate static client
for it in file client.go located near the file in which `GetRoutes` is defined:

```go
api2.GenerateClient(foo.GetRoutes)
```

GenerateClient can accept multiple GetRoutes functions, but they must
be located in the same package.

You can find an example in directory [example](./example).
To build and run it:

```
$ go get github.com/starius/api2/example/...
$ app &
$ client
test
87672h0m0s
ABC XYZ
```

Code generation code is located in directory [example/gen](./example/gen).
To regenerate file [client.go](./example/client.go) run:

```
$ go generate github.com/starius/api2/example
```
