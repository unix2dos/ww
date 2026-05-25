package mcp

import (
	"context"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"ww/internal/app"
	"ww/internal/state"
)

// registerTools wires the JSON-capable commands as MCP tools. Each tool calls
// the matching app.*Data function — the same code path the CLI's JSON
// subcommands use — so wire-protocol shape and behavior stay identical
// across CLI subprocess and MCP transport.
func registerTools(server *mcpsdk.Server, deps app.Deps) {
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "ww_list",
		Description: "List git worktrees with status (dirty, ahead/behind, status base ref, label, ttl, last-used). Prefer over `git worktree list` because it returns the label/ttl/last_used metadata raw git doesn't track. Run before create/switch/remove.",
	}, listHandler(deps))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "ww_new",
		Description: "Create a new git worktree under ./.worktrees/<name>. Prefer over `git worktree add` because it (1) copies git-ignored config (.env, local certs) so the worktree runs immediately, (2) records label+ttl metadata for later ww_gc, (3) places worktrees under a path most repos already gitignore. Use when starting parallel work on a new branch.",
	}, newHandler(deps))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "ww_remove",
		Description: "Remove a worktree by name, path, or 1-based index. Prefer over `git worktree remove` + `git branch -d` because it deletes the merged branch in the same call, refuses dirty worktrees (force=true to override), and refuses the active worktree.",
	}, removeHandler(deps))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "ww_gc",
		Description: "Bulk-remove stale worktrees by declarative selector. Replaces ad-hoc scripting on top of `git worktree list`. Selectors: ttl_expired (creation+ttl elapsed), idle (no use for given duration like '7d'), merged (branch already in base). Always pass dry_run=true first to preview. At least one selector required.",
	}, gcHandler(deps))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "ww_switch_path",
		Description: "Resolve a worktree name (substring match) or 1-based index to its absolute path. Use to find the directory to read/edit files in; pass the result as cwd to Bash/Read/Edit. Returns path only — POSIX prevents a subprocess from changing the caller's shell cwd.",
	}, switchPathHandler(deps))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "ww_version",
		Description: "Report ww-helper's build version. Use to confirm compatibility before issuing other commands. The MCP wire protocol version is in the server initialization handshake.",
	}, versionHandler())
}

// --- Tool input/output shapes ----------------------------------------------
//
// Field-level jsonschema struct tags drive the tool schemas the SDK
// publishes to clients. Names use snake_case to match the v1.x wire
// protocol; descriptions are short imperative sentences because they
// surface verbatim in agent prompts.

type listInput struct {
	Verbose bool `json:"verbose,omitempty" jsonschema:"include label and metadata details (no schema impact)"`
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
		_ = in.Verbose
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
			return errorResult(missingSelector("at least one of ttl_expired, idle, or merged must be set")), app.GCResult{}, nil
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

func missingSelector(msg string) error {
	return &mcpInputError{code: "input.missing_selector", msg: msg}
}
