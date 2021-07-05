module github.com/borzhchevskiy/go-express

go 1.16

require (
	github.com/borzhchevskiy/go-express/internal/static v0.0.0-00010101000000-000000000000
	github.com/borzhchevskiy/go-express/internal/status v0.0.0-00010101000000-000000000000
	github.com/cornelk/hashmap v1.0.1
	github.com/dlclark/regexp2 v1.4.0 // indirect
	github.com/joomcode/errorx v1.0.3
	github.com/soongo/path-to-regexp v1.6.3
)

replace github.com/borzhchevskiy/go-express => ./

replace github.com/borzhchevskiy/go-express/internal/status => ./internal/status

replace github.com/borzhchevskiy/go-express/internal/static => ./internal/static
