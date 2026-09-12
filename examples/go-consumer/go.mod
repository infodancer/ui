module github.com/infodancer/ui/examples/go-consumer

go 1.26.4

require (
	github.com/infodancer/logging v0.1.3
	github.com/infodancer/logging/httplog v0.1.1
	github.com/infodancer/ui v0.0.0
)

require github.com/felixge/httpsnoop v1.1.0 // indirect

replace github.com/infodancer/ui => ../..
