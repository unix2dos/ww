package mcp

import (
	"context"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"ww/internal/app"
	"ww/internal/state"
)

// registerTools wires the six v1.0 commands as MCP tools. Each tool calls
// the matching app.*Data function — the same code path the CLI's JSON
// subcommands use — so wire-protocol shape and behavior stay identical
// across CLI subprocess and MCP transport.
func registerTools(server *mcpsdk.Server, deps app.Deps) {
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "ww_list",
		Description: "List git worktrees in the current repository with status (dirty, ahead/behind, label, ttl). Use this to see what worktrees exist before creating, switching, or removing.",
	}, listHandler(deps))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "ww_new",
		Description: "Create a new git worktree under ./.worktrees/<name>. By default copies git-ignored files (.env etc.) from the main worktree; pass no_sync=true to skip. Returns the absolute path of the new worktree so the caller can read or write files in it.",
	}, newHandler(deps))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "ww_remove",
		Description: "Remove a worktree by name, path, or 1-based index. Refuses to remove a dirty worktree unless force=true. Refuses to remove the current worktree.",
	}, removeHandler(deps))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "ww_gc",
		Description: "Evaluate cleanup rules (ttl_expired, idle, merged) and optionally remove matched worktrees. Pass dry_run=true to only report matches. At least one of ttl_expired, idle, or merged must be set.",
	}, gcHandler(deps))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "ww_switch_path",
		Description: "Resolve a worktree name or 1-based index to its absolute path. The path is the directory the caller should read or operate inside. Note: this tool returns a path; it cannot change the caller's shell working directory (that is impossible from a subprocess).",
	}, switchPathHandler(deps))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "ww_version",
		Description: "Report the binary's build version. The MCP server's protocol version is reported in the server initialization handshake.",
	}, versionHandler())
}

// --- Tool input/output shapes ----------------------------------------------
//
// Field-level jsonschema struct tags drive the tool schemas the SDK
// publishes to clients. Names use snake_case to match the v1.0 wire
// protocol; descriptions are short imperative sentences because they
// surface verbatim in agent prompts.

type listInput struct {
	Filter  []string `json:"filter,omitempty" jsonschema:"optional filter expressions, same grammar as ww-helper list --filter (currently unstable; prefer client-side filtering)"`
	Verbose bool     `json:"verbose,omitempty" jsonschema:"include label and metadata details (no schema impact)"`
}

type listOutput struct {
	Worktrees []app.WorktreeView `json:"worktrees" jsonschema:"all worktrees matching the filter; empty array if none"`
}

type newInput struct {
	Name       string `json:"name" jsonschema:"branch name (also the directory name under .worktrees/)"`
	Label      string `json:"label,omitempty" jsonschema:"optional free-form label, e.g. 'agent:claude'"`
	TTL        string `json:"ttl,omitempty" jsonschema:"optional duration like '24h' or '7d' for ww_gc --ttl_expired"`
	Message    string `json:"message,omitempty" jsonschema:"optional task note recorded alongside the label"`
	NoSync     bool   `json:"no_sync,omitempty" jsonschema:"opt out of copying git-ignored files from the main worktree"`
	SyncDryRun bool   `json:"sync_dry_run,omitempty" jsonschema:"report what would be synced without writing files"`
}

type removeInput struct {
	Target string `json:"target" jsonschema:"worktree name, absolute path, or 1-based list index"`
	Force  bool   `json:"force,omitempty" jsonschema:"remove even if the worktree has uncommitted changes (data-loss risk)"`
}

type gcInput struct {
	TTLExpired bool   `json:"ttl_expired,omitempty" jsonschema:"match worktrees whose ttl has elapsed since creation"`
	Idle       string `json:"idle,omitempty" jsonschema:"match worktrees idle for at least this duration, e.g. '7d'"`
	Merged     bool   `json:"merged,omitempty" jsonschema:"match worktrees whose branch is already merged into the base branch"`
	DryRun     bool   `json:"dry_run,omitempty" jsonschema:"report matches without removing anything"`
	Force      bool   `json:"force,omitempty" jsonschema:"remove even dirty worktrees (otherwise they are skipped)"`
	Base       string `json:"base,omitempty" jsonschema:"override the base branch used for the merged check"`
}

type switchInput struct {
	Target string `json:"target" jsonschema:"worktree name (substring match) or 1-based list index"`
}

type versionInput struct{}

// --- Handlers --------------------------------------------------------------

func listHandler(deps app.Deps) func(context.Context, *mcpsdk.CallToolRequest, listInput) (*mcpsdk.CallToolResult, listOutput, error) {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in listInput) (*mcpsdk.CallToolResult, listOutput, error) {
		// Filter expressions are out-of-contract (protocol §6) and the
		// internal listFilter type is unexported. MVP accepts the field
		// but silently drops it; clients that need filtering should
		// fetch the full list and filter client-side, as the protocol
		// already recommends.
		_ = in.Filter

		views, _, err := app.ListData(ctx, deps, app.ListOptions{})
		if err != nil {
			return errorResult(err), listOutput{}, nil
		}
		return nil, listOutput{Worktrees: views}, nil
	}
}

func newHandler(deps app.Deps) func(context.Context, *mcpsdk.CallToolRequest, newInput) (*mcpsdk.CallToolResult, app.NewPathResult, error) {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in newInput) (*mcpsdk.CallToolResult, app.NewPathResult, error) {
		if in.Name == "" {
			return errorResult(invalidArgument("name is required")), app.NewPathResult{}, nil
		}
		result, warnings, err := app.NewPathData(ctx, deps, app.NewPathOptions{
			Name:       in.Name,
			Label:      in.Label,
			TTL:        in.TTL,
			Message:    in.Message,
			Sync:       !in.NoSync,
			SyncDryRun: in.SyncDryRun,
		})
		if err != nil {
			return errorResult(err), app.NewPathResult{}, nil
		}
		if extras := warningContents(warnings); extras != nil {
			return &mcpsdk.CallToolResult{Content: extras}, result, nil
		}
		return nil, result, nil
	}
}

func removeHandler(deps app.Deps) func(context.Context, *mcpsdk.CallToolRequest, removeInput) (*mcpsdk.CallToolResult, app.RemoveResult, error) {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in removeInput) (*mcpsdk.CallToolResult, app.RemoveResult, error) {
		result, err := app.RemoveData(ctx, deps, app.RemoveOptions{
			Target: in.Target,
			Force:  in.Force,
		})
		if err != nil {
			return errorResult(err), app.RemoveResult{}, nil
		}
		return nil, result, nil
	}
}

func gcHandler(deps app.Deps) func(context.Context, *mcpsdk.CallToolRequest, gcInput) (*mcpsdk.CallToolResult, app.GCResult, error) {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in gcInput) (*mcpsdk.CallToolResult, app.GCResult, error) {
		if !in.TTLExpired && !in.Merged && in.Idle == "" {
			return errorResult(invalidArgument("at least one of ttl_expired, idle, or merged must be set")), app.GCResult{}, nil
		}

		opts := app.GCOptions{
			TTLExpired: in.TTLExpired,
			Merged:     in.Merged,
			DryRun:     in.DryRun,
			Force:      in.Force,
			Base:       in.Base,
		}
		if in.Idle != "" {
			spec, err := state.ParseHumanDuration(in.Idle)
			if err != nil {
				return errorResult(invalidDuration(err.Error())), app.GCResult{}, nil
			}
			opts.IdleSet = true
			opts.Idle = spec
		}

		result, err := app.GCData(ctx, deps, opts)
		if err != nil {
			return errorResult(err), app.GCResult{}, nil
		}
		return nil, result, nil
	}
}

func switchPathHandler(deps app.Deps) func(context.Context, *mcpsdk.CallToolRequest, switchInput) (*mcpsdk.CallToolResult, app.SwitchPathResult, error) {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in switchInput) (*mcpsdk.CallToolResult, app.SwitchPathResult, error) {
		result, err := app.SwitchPathData(ctx, deps, in.Target)
		if err != nil {
			return errorResult(err), app.SwitchPathResult{}, nil
		}
		return nil, result, nil
	}
}

func versionHandler() func(context.Context, *mcpsdk.CallToolRequest, versionInput) (*mcpsdk.CallToolResult, app.VersionResult, error) {
	return func(_ context.Context, _ *mcpsdk.CallToolRequest, _ versionInput) (*mcpsdk.CallToolResult, app.VersionResult, error) {
		return nil, app.VersionData(), nil
	}
}

// mcpInputError carries an explicit protocol code for argument-validation
// failures that originate inside the MCP layer (before reaching app). It
// satisfies the codedError interface in translate.go so classifyForMCP
// surfaces the right code.
type mcpInputError struct {
	code string
	msg  string
}

func (e *mcpInputError) Error() string     { return e.msg }
func (e *mcpInputError) ErrorCode() string { return e.code }

func invalidArgument(msg string) error {
	return &mcpInputError{code: "input.invalid_argument", msg: msg}
}

func invalidDuration(msg string) error {
	return &mcpInputError{code: "input.invalid_duration", msg: msg}
}
