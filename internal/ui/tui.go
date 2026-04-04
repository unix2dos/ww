package ui

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"

	"golang.org/x/term"

	"ww/internal/worktree"
)

type RawMode interface {
	Prepare(in io.Reader) (restore func(), err error)
}

type OSRawMode struct{}

type tuiKey int

const (
	keyUnknown tuiKey = iota
	keyUp
	keyDown
	keyEnter
	keyCancel
)

func (OSRawMode) Prepare(in io.Reader) (func(), error) {
	file, ok := in.(*os.File)
	if !ok {
		return func() {}, nil
	}

	fd := int(file.Fd())
	if !term.IsTerminal(fd) {
		return func() {}, nil
	}

	state, err := term.MakeRaw(fd)
	if err != nil {
		return nil, err
	}
	return func() {
		_ = term.Restore(fd, state)
	}, nil
}

func RenderTUI(w io.Writer, items []worktree.Worktree, active int) {
	fmt.Fprint(w, "\x1b[H\x1b[2J")
	branchWidth := HumanBranchWidth(items)
	abWidth := aheadBehindWidth(items)
	fcWidth := fileChangesWidth(items)
	for i, item := range items {
		fmt.Fprintln(w, formatTUIRow(item, i == active, branchWidth, abWidth, fcWidth))
	}
	fmt.Fprintln(w)
	fmt.Fprint(w, "Use Up/Down (or j/k). Enter to confirm. Esc/Ctrl-C to cancel.")
}

func SelectWorktreeWithTUI(in io.Reader, out io.Writer, items []worktree.Worktree, raw RawMode) (worktree.Worktree, error) {
	if len(items) == 0 {
		return worktree.Worktree{}, errors.New("no worktrees available")
	}
	if raw == nil {
		raw = OSRawMode{}
	}

	restore, err := raw.Prepare(in)
	if err != nil {
		return worktree.Worktree{}, err
	}
	defer restore()

	reader := bufio.NewReader(in)
	active := initialActiveIndex(items)

	for {
		RenderTUI(out, items, active)

		key, err := readTUIKey(reader)
		if err != nil {
			return worktree.Worktree{}, err
		}

		switch key {
		case keyUp:
			active = (active - 1 + len(items)) % len(items)
		case keyDown:
			active = (active + 1) % len(items)
		case keyEnter:
			fmt.Fprintln(out)
			return items[active], nil
		case keyCancel:
			fmt.Fprintln(out)
			return worktree.Worktree{}, ErrSelectionCanceled
		case keyUnknown:
			// Ignore unknown keys and wait for a navigational or confirm key.
		}
	}
}

func initialActiveIndex(items []worktree.Worktree) int {
	for i, item := range items {
		if item.IsCurrent {
			return i
		}
	}
	return 0
}

func readTUIKey(reader *bufio.Reader) (tuiKey, error) {
	b, err := reader.ReadByte()
	if err != nil {
		return keyUnknown, err
	}

	switch b {
	case '\r', '\n':
		return keyEnter, nil
	case 0x03:
		return keyCancel, nil
	case 'j':
		return keyDown, nil
	case 'k':
		return keyUp, nil
	case 0x1b:
		next, err := reader.ReadByte()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return keyCancel, nil
			}
			return keyUnknown, err
		}
		if next != '[' {
			return keyCancel, nil
		}
		dir, err := reader.ReadByte()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return keyCancel, nil
			}
			return keyUnknown, err
		}
		switch dir {
		case 'A':
			return keyUp, nil
		case 'B':
			return keyDown, nil
		default:
			return keyUnknown, nil
		}
	default:
		return keyUnknown, nil
	}
}
