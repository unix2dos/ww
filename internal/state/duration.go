package state

import (
	"fmt"
	"strconv"
	"time"
)

type DurationSpec struct {
	Raw   string
	Value time.Duration
}

func ParseHumanDuration(input string) (DurationSpec, error) {
	if len(input) < 2 {
		return DurationSpec{}, fmt.Errorf("invalid duration %q", input)
	}

	unit := input[len(input)-1]
	multiplier := time.Duration(0)
	switch unit {
	case 'm':
		multiplier = time.Minute
	case 'h':
		multiplier = time.Hour
	case 'd':
		multiplier = 24 * time.Hour
	case 'w':
		multiplier = 7 * 24 * time.Hour
	default:
		return DurationSpec{}, fmt.Errorf("invalid duration %q", input)
	}

	value, err := strconv.Atoi(input[:len(input)-1])
	if err != nil || value <= 0 {
		return DurationSpec{}, fmt.Errorf("invalid duration %q", input)
	}

	return DurationSpec{
		Raw:   input,
		Value: time.Duration(value) * multiplier,
	}, nil
}

func (d DurationSpec) String() string {
	return d.Raw
}

func (d DurationSpec) ExpiresAt(createdAt time.Time) time.Time {
	return createdAt.Add(d.Value)
}
