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

A field must not have more than one of tags: `json`, `query`, `header`.
Fields in query and header parts are encoded and decoded with
`fmt.Sprintf` and `fmt.Sscanf`. Strings are not decoded with `fmt.Sscanf`,
but passed as is. Types implementing `encoding.TextMarshaler` and
`encoding.TextUnmarshaler` are encoded and decoded using it.
If no field is no JSON field in the struct, then HTTP body is skipped.

If you need the top-level type matching body JSON to be not a struct,
but of some other kind (e.g. slice or map), you should provide a field
in your struct with tag `use_as_body:"true"`:

```go
type FooRequest struct {
	// Body of the request is JSON array of strings: ["abc", "eee", ...].
	Body []string `use_as_body:"true"`

	// You can add 'header' and 'query' fields here, but not 'json'.
}
```

Now let's write the function that generates the table of routes:

```go
func GetRoutes(s *Foo) []api2.Route {
	return []api2.Route{
		{
			Method:    http.MethodPost,
			Path:      "/v1/foo/bar",
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
	...
})
if err != nil {
	panic(err)
}
// Server's response is in variable barRes.
```

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
		{Method: http.MethodPost, Path: "/v1/foo/bar", Handler: api2.Method(&s, "Bar"), Transport: &api2.JsonTransport{}},
	}
}
```

If you have function `GetRoutes` in package `foo` as above you can generate static client
for it in file client.go located near the file in which `GetRoutes` is defined:

```go
api2.GenerateClient(foo.GetRoutes)
```

You can find an example in directory [example](./example).
To build and run it:

```
$ go get github.com/starius/api2/example/...
$ server &
$ client
test
```

Code generation code is located in directory [example/gen](./example/gen).
To regenerate file [client.go](./example/client.go) run:

```
$ go generate github.com/starius/api2/example
```
