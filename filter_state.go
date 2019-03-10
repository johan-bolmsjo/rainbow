package main

import (
	"github.com/johan-bolmsjo/rainbow/internal/igor"
	"regexp"
	"strings"
)

type globalFilterState struct {
	l []*filterState
}

func (gfs *globalFilterState) clear() {
	for _, v := range gfs.l {
		v.clear()
	}
}

func (gfs *globalFilterState) allocState() *filterState {
	fs := new(filterState)
	gfs.l = append(gfs.l, fs)
	return fs
}

type filterState struct {
	matched bool // Regexp matched current line

	// Current and previously matched line.
	hist [2]struct {
		line []byte  // Line data
		res  [][]int // Regexp match result
	}
}

// match matches a line against a regexp and updates the match result. The line
// is saved for future use so it's assumed that each input line is uniquely
// allocated and not modified. The filter state is cleared after each line of
// input.
func (fs *filterState) match(line []byte, re *regexp.Regexp, updateMatched bool) [][]int {
	hist := &fs.hist[0]

	if hist.res == nil {
		if hist.res = re.FindAllSubmatchIndex(line, -1); hist.res != nil {
			hist.line = line
		}
	}

	if updateMatched && hist.res != nil {
		fs.matched = true
	}

	return hist.res
}

func (fs *filterState) clear() {
	if fs.matched {
		fs.hist[1] = fs.hist[0]
	}
	fs.matched = false
	fs.hist[0].line = nil
	fs.hist[0].res = nil
}

// valueMatchResultN returns the current or previously matched regexp result as
// a string with each regexp group separated by a zero byte marker. Regexp
// groups without a match are represented as no data but the zero marker added
// between groups.
func (fs *filterState) valueMatchResultN(n int) igor.ObjectString {
	if n < 0 || n >= len(fs.hist) {
		return igor.ObjectString("")
	}
	hist := &fs.hist[n]

	const groupSepMarker = 0

	var sb strings.Builder
	applyToRegexpResult(hist.res, func(_ int, ival interval) {
		if sb.Len() > 0 {
			sb.WriteByte(groupSepMarker)
		}
		if ival.beg != -1 {
			sb.Write(hist.line[ival.beg:ival.end])
		}
	})

	return igor.ObjectString(sb.String())
}
