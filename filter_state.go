package main

import (
	"regexp"
	"strings"

	"github.com/johan-bolmsjo/rainbow/internal/igor"
)

// globalFilterState owns the state of every filter in a program so that it can
// be reset between input lines.
type globalFilterState struct {
	l []*filterState
}

// clear clears the state of every filter.
func (gfs *globalFilterState) clear() {
	for _, v := range gfs.l {
		v.clear()
	}
}

// allocateState returns a new filter state tracked by the global state.
func (gfs *globalFilterState) allocateState() *filterState {
	fs := new(filterState)
	gfs.l = append(gfs.l, fs)
	return fs
}

// filterMatchHistorySize is the number of lines for which match results are kept: the
// current line and the previously matched line.
const filterMatchHistorySize = 2

// filterState is the match state of a filter. It keeps the match result of the
// current and the previously matched line.
type filterState struct {
	matched bool // Regexp matched current line

	// Current and previously matched line.
	hist [filterMatchHistorySize]struct {
		line []byte  // Line data
		res  [][]int // Regexp match result
	}
}

// match matches a line against a regexp and caches the match result. The line
// is saved for future use so it's assumed that each input line is uniquely
// allocated and not modified. The filter state is cleared after each line of
// input.
func (fs *filterState) match(line []byte, re *regexp.Regexp) [][]int {
	hist := &fs.hist[0]

	if hist.res == nil {
		if hist.res = re.FindAllSubmatchIndex(line, -1); hist.res != nil {
			hist.line = line
		}
	}

	return hist.res
}

// clear prepares the state for the next line. The current result is kept as the
// previous result whenever the regexp matched, also when the match was obtained
// through another filter's regexpFrom reference.
func (fs *filterState) clear() {
	if fs.hist[0].res != nil {
		fs.hist[1] = fs.hist[0]
	}
	fs.matched = false
	fs.hist[0].line = nil
	fs.hist[0].res = nil
}

// valueMatchResult returns the current or previously matched regexp result as
// a string with each regexp group separated by a zero byte marker. Regexp
// groups without a match are represented as no data but the zero marker added
// between groups.
func (fs *filterState) valueMatchResult(n int) igor.ObjectString {
	if n < 0 || n >= len(fs.hist) {
		return igor.ObjectString("")
	}
	hist := &fs.hist[n]

	const groupSepMarker = 0

	var sb strings.Builder
	groupCount := 0
	applyToRegexpResult(hist.res, func(_ int, ival interval) {
		// Write the group separator before every group except the first.
		// Groups without a match contribute no data but still take part in
		// the separation.
		if groupCount > 0 {
			sb.WriteByte(groupSepMarker)
		}
		groupCount++
		if ival.beg != -1 {
			sb.Write(hist.line[ival.beg:ival.end])
		}
	})

	return igor.ObjectString(sb.String())
}
