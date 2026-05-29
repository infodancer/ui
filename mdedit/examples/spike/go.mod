module github.com/infodancer/ui/mdedit/examples/spike

go 1.26.3

require (
	github.com/infodancer/ui v0.0.0-20260519215753-aee89b6d5ada
	github.com/infodancer/ui/markdown v0.2.0
	github.com/infodancer/ui/mdedit v0.1.0
)

// In-repo example: resolve the sibling modules from the tree. The required
// versions above are the tags Phase 1 publishes; these replaces let the example
// build from a checkout before/independent of those tags.
replace (
	github.com/infodancer/ui => ../../..
	github.com/infodancer/ui/markdown => ../../../markdown
	github.com/infodancer/ui/mdedit => ../..
)

require (
	github.com/aymerick/douceur v0.2.0 // indirect
	github.com/gorilla/css v1.0.1 // indirect
	github.com/microcosm-cc/bluemonday v1.0.27 // indirect
	github.com/yuin/goldmark v1.8.2 // indirect
	golang.org/x/net v0.55.0 // indirect
)
