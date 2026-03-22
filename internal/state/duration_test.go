package state

import (
	"testing"
	"time"
)

func TestParseHumanDurationAcceptsSupportedUnits(t *testing.T) {
	t.Parallel()

	cases := []struct {
		input       string
		wantString  string
		wantValue   time.Duration
	}{
		{input: "15m", wantString: "15m", wantValue: 15 * time.Minute},
		{input: "24h", wantString: "24h", wantValue: 24 * time.Hour},
		{input: "7d", wantString: "7d", wantValue: 7 * 24 * time.Hour},
		{input: "2w", wantString: "2w", wantValue: 14 * 24 * time.Hour},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()

			got, err := ParseHumanDuration(tc.input)
			if err != nil {
				t.Fatalf("ParseHumanDuration(%q) error: %v", tc.input, err)
			}
			if got.String() != tc.wantString {
				t.Fatalf("String() = %q, want %q", got.String(), tc.wantString)
			}
			if got.Value != tc.wantValue {
				t.Fatalf("Value = %v, want %v", got.Value, tc.wantValue)
			}
		})
	}
}

func TestParseHumanDurationRejectsInvalidValues(t *testing.T) {
	t.Parallel()

	for _, input := range []string{"", "7", "1mo", "abc", "0h", "-2d"} {
		input := input
		t.Run(input, func(t *testing.T) {
			t.Parallel()

			if _, err := ParseHumanDuration(input); err == nil {
				t.Fatalf("ParseHumanDuration(%q) succeeded, want error", input)
			}
		})
	}
}

func TestDurationSpecExpiry(t *testing.T) {
	t.Parallel()

	spec, err := ParseHumanDuration("24h")
	if err != nil {
		t.Fatalf("ParseHumanDuration returned error: %v", err)
	}

	createdAt := time.Unix(100, 0)
	got := spec.ExpiresAt(createdAt)
	want := createdAt.Add(24 * time.Hour)
	if !got.Equal(want) {
		t.Fatalf("ExpiresAt() = %v, want %v", got, want)
	}
}
