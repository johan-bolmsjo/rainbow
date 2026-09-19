package main

import (
	"regexp"
	"strings"
	"testing"
)

// TestZeroLengthGroupMatchDoesNotPanic verifies that an optional regexp group
// that matches the empty string is handled without error.
func TestZeroLengthGroupMatchDoesNotPanic(t *testing.T) {
	const config = `{
  filter: {
    name: example
    regexp: ` + "`(a)(b?)c`" + `
    properties: {
      1: { color: red }
      2: { color: green }
    }
  }
  apply: { filters: example }
}`

	testProgram, err := createProgram(strings.NewReader(config))
	if err != nil {
		t.Fatalf("createProgram: %v", err)
	}

	for _, text := range []string{"ac", "abc"} {
		l := newLine()
		l.init([]byte(text))
		if err := l.applyProgram(testProgram); err != nil {
			t.Fatalf("applyProgram(%q): %v", text, err)
		}
	}
}

// TestPropertyGroupOutOfRange verifies that properties referring to a
// non-existing regexp group are rejected.
func TestPropertyGroupOutOfRange(t *testing.T) {
	const config = `{
  filter: {
    name: example
    regexp: ` + "`(a)`" + `
    properties: {
      2: { color: red }
    }
  }
  apply: { filters: example }
}`

	if _, err := createProgram(strings.NewReader(config)); err == nil {
		t.Fatal("expected error for out-of-range regexp group")
	}
}

// TestMatchResultEmptyLeadingGroup verifies that a group separator is written
// before an empty group that precedes a non-empty group.
func TestMatchResultEmptyLeadingGroup(t *testing.T) {
	state := &filterState{}
	state.match([]byte("y"), regexp.MustCompile(`(x)?(y)`))

	if got, want := string(state.valueMatchResult(0)), "\x00y"; got != want {
		t.Fatalf("match result = %q, want %q", got, want)
	}
}

// TestMatchResultOutOfRangeIndex verifies that a match result for an out of
// range index is reported as an empty string.
func TestMatchResultOutOfRangeIndex(t *testing.T) {
	state := &filterState{}

	for _, n := range []int{-1, 2} {
		if got := string(state.valueMatchResult(n)); got != "" {
			t.Errorf("valueMatchResult(%d) = %q, want empty", n, got)
		}
	}
}
