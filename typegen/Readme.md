# Golang structs to typescript typings convertor

## Example

[example.main.go](https://github.com/zmitry/go2typings/blob/master/example/main.go)

## How to setup

- create go file with the code bellow
- run this code with `go run`

```golang
package main

import (
  "github.com/zmitry/go2ts"
   // you can use your own
  "github.com/zmitry/go2ts/example/types"
)

type Root struct {
	User types.User
	T    types.T
}

func main() {
	s := go2ts.New()
	s.Add(types.T{})
	s.Add(types.User{})

	err := s.GenerateFile("./test.ts")
	if err != nil {
		panic(err)
	}
}
```

# Custom tags

we support custom tag `ts` it has the following syntax

```
type M struct {
	Username string `json:"Username2" ts:"string,optional"`
}
```

tsTag type

```
tsTag[0] = "string"|"date"|"-"
tsTag[1] = "optional"|"no-null"|"null"
```

see field.go for more info
