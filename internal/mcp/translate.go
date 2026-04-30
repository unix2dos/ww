package mcp

import (
	"encoding/json"
	"errors"
	"fmt"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"ww/internal/app"
)

// errorResult converts an app-layer error into the MCP CallToolResult shape
// that signals a tool-level failure. The result's Content carries a single
// JSON line in the same shape as the CLI envelope's `error` object so the
// agent can branch on `code` (e.g. "worktree.dirty") just like it would
// against `ww-helper --json`.
func errorResult(err error) *mcpsdk.CallToolResult {
	code, message := classifyForMCP(err)
	payload := map[string]any{
		"code":    code,
		"message": message,
		"context": map[string]any{},
	}
	encoded, _ := json.Marshal(payload)
	return &mcpsdk.CallToolResult{
		IsError: true,
		Content: []mcpsdk.Content{
			&mcpsdk.TextContent{Text: string(encoded)},
		},
	}
}

// warningContent renders a single warning entry as an MCP TextContent block.
// The format is one JSON line per warning, mirroring the wire-protocol
// shape so agents can parse them with the same code path that handles the
// CLI's `warnings` array.
func warningContents(warnings []app.Warning) []mcpsdk.Content {
	if len(warnings) == 0 {
		return nil
	}
	contents := make([]mcpsdk.Content, 0, len(warnings))
	for _, w := range warnings {
		payload := map[string]any{
			"warning": w,
		}
		encoded, _ := json.Marshal(payload)
		contents = append(contents, &mcpsdk.TextContent{Text: string(encoded)})
	}
	return contents
}

// codedError is implemented by any error that carries a stable protocol
// code. Both the MCP layer's mcpInputError and (via app.ErrorCode) the app
// layer's typed errors satisfy the contract from the caller's point of view.
type codedError interface {
	error
	ErrorCode() string
}

// classifyForMCP returns (code, message) for an error surfaced from a tool
// handler. It checks the codedError interface first (to catch MCP-layer
// errors), then falls through to app.ErrorCode for app-layer typed errors.
// Plain errors map to a generic git/command failure.
func classifyForMCP(err error) (string, string) {
	if err == nil {
		return "", ""
	}
	var c codedError
	if errors.As(err, &c) {
		return c.ErrorCode(), err.Error()
	}
	if code := app.ErrorCode(err); code != "" {
		return code, err.Error()
	}
	return "git.command_failed", fmt.Sprintf("%v", err)
}
