package main

import (
	"context"
	"os"

	"ww/internal/app"
	"ww/internal/mcp"
)

func init() {
	app.MCPServe = mcp.Serve
}

func main() {
	os.Exit(app.Run(context.Background(), os.Args[1:], os.Stdin, os.Stdout, os.Stderr, app.RealDeps{}))
}
