/*
Package api2 provides types and functions used to define interfaces of
client-server API and facilitate creation of server and client for it.

How to use this package. Organize your code in services. Each service provides
some domain specific functionality. It is a Go type whose methods correspond
to exposed RPC's of the API. Each method has the following signature:

	func(ctx, *Request) (*Response, error)

Let's define a service Foo with method Bar.

	type Foo struct {
		...
	}

	type BarRequest struct {
		...
	}

	type BarResponse struct {
		...
	}

	func (s *Foo) Bar(ctx context.Context, req *BarRequest) (*BarResponse, error) {
		...
	}

Now let's write the function that generates the table of routes:

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

You can add multiple routes with the same path, but in this case their
HTTP methods must be different so that they can be distinguished.

In the server you need a real instance of service Foo to pass to GetRoutes.
Then just bind the routes to http.ServeMux and run the server:

	// Server.
	foo := NewFoo(...)
	routes := GetRoutes(foo)
	api2.BindRoutes(http.DefaultServeMux, routes)
	log.Fatal(http.ListenAndServe(":8080", nil))

The server is running.
It serves foo.Bar function on path /v1/foo/bar with HTTP method Post.

Now let's create the client:

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

Note that you don't have to pass a real service object to GetRoutes
on client side. You can pass nil, it is sufficient to pass all needed
information about request and response types in the routes table, that
is used by client to find a proper route.

You can make GetRoutes accepting an interface instead of a concrete
Service type. In this case you can not get method handlers by s.Bar,
because this code panics if s is nil interface. As a workaround api2
provides function Method(service pointer, methodName) which you can use:

	type Service interface {
		Bar(ctx context.Context, req *BarRequest) (*BarResponse, error)
	}

	func GetRoutes(s Service) []api2.Route {
		return []api2.Route{
			{Method: http.MethodPost, Path: "/v1/foo/bar", Handler: api2.Method(&s, "Bar"), Transport: &api2.JsonTransport{}},
		}
	}
*/
package api2
