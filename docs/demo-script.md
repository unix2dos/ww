# ww Demo Script Notes

This file keeps the demo storyline stable when `ww` changes again.

## Storyline

The canonical demo is a short `switch -> new -> rm` loop for first-time users:

1. Start in `main` inside a throwaway repository.
2. Run `ww` and use the `fzf` fast path to select `feat-a`.
3. Confirm the shell is now on `feat-a`.
4. Run `ww new feat-demo` and confirm the shell moved into the new worktree.
5. Switch back to `main`.
6. Run `ww rm feat-demo`, confirm the prompt, and show the happy-path branch deletion output.

The generator installs `scripts/demo-fzf.sh` as a deterministic `fzf` shim so the recording stays stable across machines while still exercising the `fzf` code path.

## Pacing Knobs

`bash scripts/generate-demo.sh` accepts these environment overrides:

- `WW_DEMO_KEYSTROKE_DELAY_MS`
- `WW_DEMO_STEP_DELAY_MS`
- `WW_DEMO_FZF_FOCUS_DELAY_MS`
- `WW_DEMO_FZF_QUERY_SETTLE_MS`
- `WW_DEMO_CONFIRM_DELAY_MS`
- `WW_DEMO_IDLE_TIME_LIMIT`

The default pacing is tuned for a `25-40s` viewing window on the Pages player.

## Regeneration

```bash
bash scripts/generate-demo.sh
```

That command rebuilds the helper, records `docs/assets/ww-demo.cast`, and regenerates `docs/assets/ww-demo.svg`.
