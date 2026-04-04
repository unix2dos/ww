package ui

import "testing"

func TestBoldWrapsTextInBoldEscapes(t *testing.T) {
	got := Bold("hello")
	if got != "\x1b[1mhello\x1b[0m" {
		t.Fatalf("expected bold escape, got %q", got)
	}
}

func TestGreenWrapsTextInGreenEscapes(t *testing.T) {
	got := Green("hello")
	if got != "\x1b[32mhello\x1b[0m" {
		t.Fatalf("expected green escape, got %q", got)
	}
}

func TestYellowWrapsTextInYellowEscapes(t *testing.T) {
	got := Yellow("hello")
	if got != "\x1b[33mhello\x1b[0m" {
		t.Fatalf("expected yellow escape, got %q", got)
	}
}

func TestRedWrapsTextInRedEscapes(t *testing.T) {
	got := Red("hello")
	if got != "\x1b[31mhello\x1b[0m" {
		t.Fatalf("expected red escape, got %q", got)
	}
}

func TestDimWrapsTextInDimEscapes(t *testing.T) {
	got := Dim("hello")
	if got != "\x1b[2mhello\x1b[0m" {
		t.Fatalf("expected dim escape, got %q", got)
	}
}

func TestEmptyStringReturnsEmpty(t *testing.T) {
	if got := Green(""); got != "" {
		t.Fatalf("expected empty for empty input, got %q", got)
	}
}

func TestStripAnsiRemovesEscapeSequences(t *testing.T) {
	colored := Bold("hello")
	got := StripAnsi(colored)
	if got != "hello" {
		t.Fatalf("expected stripped text, got %q", got)
	}
}

func TestVisualLenReturnsLengthWithoutAnsi(t *testing.T) {
	colored := Green("hello")
	if got := VisualLen(colored); got != 5 {
		t.Fatalf("expected visual len 5, got %d", got)
	}
}
