module github.com/borzhchevskiy/balda

go 1.16

require (
	github.com/borzhchevskiy/balda/internal/status v0.0.0
	github.com/cornelk/hashmap v1.0.1
	github.com/dlclark/regexp2 v1.4.0 // indirect
	github.com/joomcode/errorx v1.0.3
	github.com/soongo/path-to-regexp v1.6.3
)

replace github.com/borzhchevskiy/balda => ./

replace github.com/borzhchevskiy/balda/internal/status => ./internal/status
