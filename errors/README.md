# errors wrappers

This package provides convenience wrappers for errors.

Each function in the package behaves like [fmt.Errorf](https://pkg.go.dev/fmt#Errorf) function,
attaching corresponding HTTP status. The mapping of HTTP statuses:

| Function           | HTTP code                 |
|--------------------|---------------------------|
| Aborted            | 409 Conflict              |
| AlreadyExists      | 409 Conflict              |
| Canceled           | 499 Client Closed Request |
| DataLoss           | 500 Internal Server Error |
| DeadlineExceeded   | 504 Gateway Timeout       |
| FailedPrecondition | 400 Bad Request           |
| Internal           | 500 Internal Server Error |
| InvalidArgument    | 400 Bad Request           |
| NotFound           | 404 Not Found             |
| OutOfRange         | 400 Bad Request           |
| PermissionDenied   | 403 Forbidden             |
| ResourceExhausted  | 429 Too Many Requests     |
| Unauthenticated    | 401 Unauthorized          |
| Unavailable        | 503 Service Unavailable   |
| Unimplemented      | 501 Not Implemented       |
| Unknown            | 500 Internal Server Error |

The list can be found [here](https://github.com/googleapis/googleapis/blob/master/google/rpc/code.proto).
