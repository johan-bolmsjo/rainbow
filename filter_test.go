package main

import (
	"fmt"
	"strings"
	"testing"
)

// renderedSegment is a line segment together with the filter properties that
// were applied to it.
type renderedSegment struct {
	beg, end int
	text     string
	props    properties
}

// renderedLine is a processed input line split into rendered segments.
type renderedLine struct {
	text     string
	segments []renderedSegment
}

// segmentTexts returns the text of each segment in order.
func (rl renderedLine) segmentTexts() []string {
	texts := make([]string, len(rl.segments))
	for i, s := range rl.segments {
		texts[i] = s.text
	}
	return texts
}

// segmentContaining returns the segment that contains the first occurrence of
// text. It is used when the properties of interest are merged with neighboring
// text instead of forming a segment of their own.
func (rl renderedLine) segmentContaining(t *testing.T, text string) renderedSegment {
	t.Helper()
	idx := strings.Index(rl.text, text)
	if idx < 0 {
		t.Fatalf("text %q not found in line %q", text, rl.text)
	}
	for _, s := range rl.segments {
		if s.beg <= idx && idx < s.end {
			return s
		}
	}
	t.Fatalf("no segment contains %q in %v", text, rl.segmentTexts())
	return renderedSegment{}
}

// newProperties returns filter properties for the given foreground color,
// background color and modifiers.
func newProperties(fgcolor, bgcolor color, modifiers ...modifier) properties {
	var props properties
	props.fgcolor = fgcolor
	props.bgcolor = bgcolor
	for _, m := range modifiers {
		props.modifiers.set(m)
	}
	return props
}

// applyProgramLines applies lines to prog in order and returns the rendered
// result of the last line. A sequence is required by filters whose conditions
// compare the current match result with the previous one.
func applyProgramLines(t *testing.T, prog *program, lines ...string) renderedLine {
	t.Helper()
	var last *line
	for _, text := range lines {
		l := newLine()
		l.init([]byte(text))
		if err := l.applyProgram(prog); err != nil {
			t.Fatalf("applyProgram(%q): %v", text, err)
		}
		last = l
	}

	segments := make([]renderedSegment, 0)
	for s := last.segmentList.Next(); s != &last.segmentList; s = s.Next() {
		segments = append(segments, renderedSegment{
			beg:   s.Value.ival.beg,
			end:   s.Value.ival.end,
			text:  string(last.text[s.Value.ival.beg:s.Value.ival.end]),
			props: s.Value.props,
		})
	}
	return renderedLine{text: string(last.text), segments: segments}
}

// applyConfiguration parses a configuration and applies it to lines in order,
// returning the rendered result of the last line.
func applyConfiguration(t *testing.T, config string, lines ...string) renderedLine {
	t.Helper()
	prog, err := createProgram(strings.NewReader(config))
	if err != nil {
		t.Fatalf("createProgram: %v", err)
	}
	return applyProgramLines(t, prog, lines...)
}

// TestFilterProperties verifies that the color, bgcolor and modifiers
// properties are applied to the configured regexp group.
func TestFilterProperties(t *testing.T) {
	tests := []struct {
		name       string
		properties string
		want       properties
	}{
		{"color", "color: red", newProperties(colorRed, colorNone)},
		{"bgcolor", "bgcolor: blue", newProperties(colorNone, colorBlue)},
		{"color and bgcolor", "color: white bgcolor: red", newProperties(colorWhite, colorRed)},
		{"modifier", "modifiers: bold", newProperties(colorNone, colorNone, modifierBold)},
		{"modifier list", "modifiers: [bold underline]",
			newProperties(colorNone, colorNone, modifierBold, modifierUnderline)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := fmt.Sprintf(`{
    filter: {
        name: test
        regexp: (match)
        properties: { 1: { %s } }
    }
    apply: { filters: test }
}`, tt.properties)

			got := applyConfiguration(t, config, "a match here").segmentContaining(t, "match")
			if got.props != tt.want {
				t.Errorf("match segment properties = %+v, want %+v", got.props, tt.want)
			}
		})
	}
}

// TestFilterPropertyGroupTargeting verifies that properties only apply to the
// regexp group they are configured for.
func TestFilterPropertyGroupTargeting(t *testing.T) {
	config := fmt.Sprintf(`{
    filter: {
        name: test
        regexp: %s
        properties: { 2: { color: red } }
    }
    apply: { filters: test }
}`, "`(ERROR|WARN): (\\w+)`")

	line := applyConfiguration(t, config, "ERROR: failure")

	if got := line.segmentContaining(t, "ERROR"); got.props.fgcolor != colorNone {
		t.Errorf("group 1 segment %q has foreground color %s, want none", got.text, got.props.fgcolor)
	}
	if got := line.segmentContaining(t, "failure"); got.props.fgcolor != colorRed {
		t.Errorf("group 2 segment %q has foreground color %s, want red", got.text, got.props.fgcolor)
	}
}

// TestNestedFilters verifies that applying a filter also applies its nested
// filters.
func TestNestedFilters(t *testing.T) {
	const config = `{
    filter: {
        name: parent
        regexp: (parent)
        properties: { 1: { color: red } }
        filter: {
            name: child
            regexp: (child)
            properties: { 1: { color: green } }
        }
    }
    apply: { filters: parent }
}`

	line := applyConfiguration(t, config, "parent and child")

	if got := line.segmentContaining(t, "parent"); got.props.fgcolor != colorRed {
		t.Errorf("parent segment %q has foreground color %s, want red", got.text, got.props.fgcolor)
	}
	if got := line.segmentContaining(t, "child"); got.props.fgcolor != colorGreen {
		t.Errorf("child segment %q has foreground color %s, want green", got.text, got.props.fgcolor)
	}
}

// TestFilterNameReference verifies that a nested filter can be referenced by
// its path using '/' as separator, both from an apply filters list and from a
// condition evaluating filter-match?.
func TestFilterNameReference(t *testing.T) {
	const filterListConfig = `{
    filter: {
        name: parent
        filter: {
            name: child
            regexp: (child)
            properties: { 1: { color: green } }
        }
    }
    apply: { filters: parent/child }
}`

	const conditionConfig = `{
    filter: {
        name: parent
        filter: {
            name: info
            regexp: (INFO)
        }
        filter: {
            name: debug
            regexp: (DEBUG)
        }
    }
    filter: {
        name: subject
        regexp: (subject)
        properties: { 1: { color: green } }
    }
    apply: { filters: parent }
    apply: { cond: [not [filter-match? parent/info parent/debug]] filters: subject }
}`

	tests := []struct {
		name   string
		config string
		line   string
		text   string
		want   color
	}{
		{"apply filters list", filterListConfig, "a child", "child", colorGreen},
		{"condition without info or debug", conditionConfig, "a subject", "subject", colorGreen},
		{"condition with info", conditionConfig, "INFO about a subject", "subject", colorNone},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := applyConfiguration(t, tt.config, tt.line).segmentContaining(t, tt.text)
			if got.props.fgcolor != tt.want {
				t.Errorf("segment %q has foreground color %s, want %s",
					got.text, got.props.fgcolor, tt.want)
			}
		})
	}
}

// TestRegexpFrom verifies that a filter can reuse the regexp and match result
// of another named filter.
func TestRegexpFrom(t *testing.T) {
	const config = `{
    filter: { name: base regexp: (foo) }
    filter: {
        name: derived
        regexpFrom: base
        properties: { 1: { color: red } }
    }
    apply: { filters: [base derived] }
}`

	line := applyConfiguration(t, config, "foo bar foo")
	if got := line.segmentContaining(t, "foo"); got.props.fgcolor != colorRed {
		t.Errorf("matched segment %q has foreground color %s, want red", got.text, got.props.fgcolor)
	}
	if got := line.segmentContaining(t, "bar"); got.props.fgcolor != colorNone {
		t.Errorf("unmatched segment %q has foreground color %s, want none", got.text, got.props.fgcolor)
	}
}

// TestRegexpFromMatchHistory verifies that a filter that is only referenced
// through another filter's regexpFrom parameter records its match result as the
// previous result for the next line.
func TestRegexpFromMatchHistory(t *testing.T) {
	const config = `{
    filter: { name: base regexp: ` + "`(x)`" + ` }
    filter: { name: derived regexpFrom: base properties: { 1: { color: red } } }
    apply: { filters: derived }
}`

	tests := []struct {
		name  string
		lines []string
		want  string
	}{
		{"match is recorded as previous", []string{"x"}, "x"},
		{"previous match is retained on mismatch", []string{"x", "y"}, "x"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prog, err := createProgram(strings.NewReader(config))
			if err != nil {
				t.Fatalf("createProgram: %v", err)
			}
			applyProgramLines(t, prog, tt.lines...)

			base := prog.findFilter("base")
			if got := string(base.state.valueMatchResult(1)); got != tt.want {
				t.Errorf("previous match result = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestFilterOverlappingIntervals verifies that properties applied to an
// interval that spans several existing segments are spliced into all of them.
func TestFilterOverlappingIntervals(t *testing.T) {
	const config = `{
    filter: { name: first regexp: (ab) properties: { 1: { color: red } } }
    filter: { name: second regexp: (abcd) properties: { 1: { bgcolor: blue } } }
    apply: { filters: [first second] }
}`

	line := applyConfiguration(t, config, "abcd")

	if got := line.segmentContaining(t, "a"); got.props != newProperties(colorRed, colorBlue) {
		t.Errorf("segment %q properties = %+v, want red foreground and blue background",
			got.text, got.props)
	}
	if got := line.segmentContaining(t, "c"); got.props != newProperties(colorNone, colorBlue) {
		t.Errorf("segment %q properties = %+v, want blue background only", got.text, got.props)
	}
}

// TestSplicePropertiesZeroLengthInterval verifies that a zero-length interval
// leaves the line unchanged.
func TestSplicePropertiesZeroLengthInterval(t *testing.T) {
	l := newLine()
	l.init([]byte("abc"))

	l.spliceProperties(interval{1, 1}, newProperties(colorRed, colorNone))

	count := 0
	for s := l.segmentList.Next(); s != &l.segmentList; s = s.Next() {
		count++
	}
	if count != 1 {
		t.Errorf("segment count = %d, want 1", count)
	}
}
