// Package mcp exposes ww-helper's v1.0 wire protocol as MCP tools so any
// MCP-aware agent (Claude Code, Cursor, Zed, Continue, Cline, Codex, …)
// can call ww-helper natively over stdio without subprocess marshalling.
//
// The package's only public function is Serve. It is invoked from the
// `ww-helper mcp serve` subcommand in package app and runs until the client
// disconnects.
package mcp

import (
	"context"
	"io"
	"log"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"ww/internal/app"
)

// Serve runs an MCP server over stdio. It returns when the client
// disconnects or ctx is cancelled.
//
// stderr is the log destination for any internal SDK or handler logging;
// stdout is reserved exclusively for the MCP JSON-RPC transport, so callers
// must NOT write anything to stdout while Serve is running.
func Serve(ctx context.Context, deps app.Deps, binaryVersion string, stderr io.Writer) error {
	// Belt-and-braces: route the standard logger to stderr in case any
	// dependency writes to it. The SDK already separates its own logging,
	// but downstream handlers calling app.* may not.
	log.SetOutput(stderr)

	server := mcpsdk.NewServer(&mcpsdk.Implementation{
		Name:    "ww",
		Version: binaryVersion,
	}, nil)

	registerTools(server, deps)

	return server.Run(ctx, &mcpsdk.StdioTransport{})
}
