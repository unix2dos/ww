package ui

import (
	"regexp"
	"strings"
)

var ansiPattern = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func wrap(code, s string) string {
	if s == "" {
		return ""
	}
	return "\x1b[" + code + "m" + s + "\x1b[0m"
}

func Bold(s string) string      { return wrap("1", s) }
func Green(s string) string     { return wrap("32", s) }
func Yellow(s string) string    { return wrap("33", s) }
func Red(s string) string       { return wrap("31", s) }
func Dim(s string) string { return wrap("2", s) }

func StripAnsi(s string) string {
	return ansiPattern.ReplaceAllString(s, "")
}

func VisualLen(s string) int {
	return len(StripAnsi(s))
}

// PadRight pads s to width based on visual (non-ANSI) length.
func PadRight(s string, width int) string {
	vl := VisualLen(s)
	if vl >= width {
		return s
	}
	return s + strings.Repeat(" ", width-vl)
}
