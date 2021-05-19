## Quickstart

```go
package main

import (
	Express "github.com/borzhchevskiy/go-express"
)

func main() {
	App := Express.Express("localhost", 8080)

	App.Get("/", func(req *Express.Request, res *Express.Response){
		res.Send("<h1>Hello World!</h1>")
	})

	App.Listen()
}
```

## Cookies

```go
App.Get("/add/", func(req *Express.Request, res *Express.Response){
	res.SetCookie(&Express.Cookie {
		Name: "Hello",
		Value: "World",
	})
	res.Send("Cookie added")
})

App.Get("/del/", func(req *Express.Request, res *Express.Response){
	res.DelCookie("Hello")
	res.Send("Cookie deleted")
})
```

## Static files

```go
App.Static("/css", "./static/css")
App.Static("/html", "./static/html")
```

## Installation

```bash
go get -u github.com/borzhchevskiy/go-express
```