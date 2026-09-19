package main

import (
	"strings"
	"testing"
)

// TestApplyFilters verifies that every filter named in an apply filters list
// is applied to the line.
func TestApplyFilters(t *testing.T) {
	const config = `{
    filter: { name: first regexp: (foo) properties: { 1: { color: red } } }
    filter: { name: second regexp: (foo) properties: { 1: { bgcolor: blue } } }
    apply: { filters: [first second] }
}`

	got := applyConfiguration(t, config, "foo").segmentContaining(t, "foo")
	if want := newProperties(colorRed, colorBlue); got.props != want {
		t.Errorf("segment %q properties = %+v, want %+v", got.text, got.props, want)
	}
}

// TestApplyCondition verifies the condition expression functions used to
// decide whether an apply clause runs. The condition combines filter-match?,
// and, not, equal? and filter-result to only highlight a word when it differs
// from the previous line.
func TestApplyCondition(t *testing.T) {
	const config = `{
    filter: { name: word regexp: ` + "`(word\\d)`" + ` }
    filter: {
        name: wordHighlight
        regexpFrom: word
        properties: { 1: { color: red } }
    }
    apply: { filters: word }
    apply: {
        cond: [and [filter-match? word]
                  [not [equal? [filter-result word 0] [filter-result word 1]]]]
        filters: wordHighlight
    }
}`

	tests := []struct {
		name  string
		lines []string
		text  string
		want  color
	}{
		{"first match is highlighted", []string{"word1"}, "word1", colorRed},
		{"repeated match is not highlighted", []string{"word1", "word1"}, "word1", colorNone},
		{"changed match is highlighted", []string{"word1", "word2"}, "word2", colorRed},
		{"mismatch is not highlighted", []string{"word1", "nothing"}, "nothing", colorNone},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := applyConfiguration(t, config, tt.lines...).segmentContaining(t, tt.text)
			if got.props.fgcolor != tt.want {
				t.Errorf("segment %q has foreground color %s, want %s",
					got.text, got.props.fgcolor, tt.want)
			}
		})
	}
}

// TestApplyConditionRegexpFromHistory verifies that the match history of a
// filter that is only referenced through regexpFrom is promoted, so that a
// condition can detect a changed match between the current and the previous
// line.
func TestApplyConditionRegexpFromHistory(t *testing.T) {
	const config = `{
    filter: { name: base regexp: ` + "`(word\\d)`" + ` }
    filter: { name: derived regexpFrom: base }
    filter: { name: highlight regexpFrom: base properties: { 1: { color: red } } }
    apply: { filters: derived }
    apply: {
        cond: [not [equal? [filter-result base 0] [filter-result base 1]]]
        filters: highlight
    }
}`

	tests := []struct {
		name  string
		lines []string
		text  string
		want  color
	}{
		{"first match is highlighted", []string{"word1"}, "word1", colorRed},
		{"repeated match is not highlighted", []string{"word1", "word1"}, "word1", colorNone},
		{"changed match is highlighted", []string{"word1", "word2"}, "word2", colorRed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := applyConfiguration(t, config, tt.lines...).segmentContaining(t, tt.text)
			if got.props.fgcolor != tt.want {
				t.Errorf("segment %q has foreground color %s, want %s",
					got.text, got.props.fgcolor, tt.want)
			}
		})
	}
}

// TestApplyConditionRegexpFromMatchStatus verifies that filter-match? reports a
// match for a filter that matched through its regexpFrom regexp.
func TestApplyConditionRegexpFromMatchStatus(t *testing.T) {
	const config = `{
    filter: { name: base regexp: ` + "`(x)`" + ` }
    filter: { name: derived regexpFrom: base }
    filter: { name: highlight regexp: ` + "`(highlight)`" + ` properties: { 1: { color: red } } }
    apply: { filters: derived }
    apply: { cond: [filter-match? derived] filters: highlight }
}`

	got := applyConfiguration(t, config, "x highlight").segmentContaining(t, "highlight")
	if got.props.fgcolor != colorRed {
		t.Errorf("segment %q has foreground color %s, want red", got.text, got.props.fgcolor)
	}
}

// TestApplyConditionErrors verifies that errors raised while evaluating an
// apply condition are reported when a line is processed.
func TestApplyConditionErrors(t *testing.T) {
	const filter = `{filter: {name: f regexp: (x)} `

	tests := []struct {
		name    string
		apply   string
		wantErr string
	}{
		{"filter-match missing filter", `{cond: [filter-match? missing] filters: f}}`, "missing filter"},
		{"filter-match type error", `{cond: [filter-match? [filter-result f abc]] filters: f}}`, "type error"},
		{"filter-result argument count", `{cond: [filter-result f] filters: f}}`, "invalid number of arguments"},
		{"filter-result missing filter", `{cond: [filter-result missing 0] filters: f}}`, "missing filter"},
		{"filter-result type error", `{cond: [filter-result f [filter-result f abc]] filters: f}}`, "type error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prog, err := createProgram(strings.NewReader(filter + "apply: " + tt.apply))
			if err != nil {
				t.Fatalf("createProgram: %v", err)
			}

			l := newLine()
			l.init([]byte("x"))
			err = l.applyProgram(prog)
			if err == nil {
				t.Fatal("expected an error")
			} else if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want it to contain %q", err, tt.wantErr)
			}
		})
	}
}

// TestApplyProgramClearsStateOnConditionError verifies that a condition error
// does not leave a cached match result behind. A stale result would be reused
// when the next, possibly shorter, line is processed.
func TestApplyProgramClearsStateOnConditionError(t *testing.T) {
	const config = `{
    filter: { name: f regexp: (x+) properties: { 1: { color: red } } }
    apply: { filters: f }
    apply: { cond: [filter-result missing 0] filters: f }
}`

	prog, err := createProgram(strings.NewReader(config))
	if err != nil {
		t.Fatalf("createProgram: %v", err)
	}

	l := newLine()
	l.init([]byte("xxxxxxxxxx"))
	if err := l.applyProgram(prog); err == nil {
		t.Fatal("expected an error for the first line")
	}

	for _, state := range prog.globalFilterState.l {
		if state.hist[0].res != nil {
			t.Fatalf("filter state not cleared after error: hist[0].res = %v", state.hist[0].res)
		}
	}

	// The short line must not trip the splice assertions by reusing the match
	// result cached for the long line.
	l.init([]byte("a"))
	if err := l.applyProgram(prog); err == nil {
		t.Fatal("expected an error for the second line")
	}
}
