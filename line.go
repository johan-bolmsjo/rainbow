package main

import (
	"io"

	"github.com/johan-bolmsjo/gods/v4/avltree"
	"github.com/johan-bolmsjo/gods/v4/list"
	"github.com/johan-bolmsjo/gods/v4/math"
)

// line is an input line and the properties applied to its byte intervals. A
// line is reused between input lines by calling init.
type line struct {
	text         []byte // shared data, must not be modified after initialization
	segmentIndex *avltree.Tree[int, *lineSegment]
	segmentList  lineSegment
}

// lineSegment is a node in the linked list of line segments. The segments are
// kept in ascending order.
type lineSegment = list.Node[lineSegmentData]

// lineSegmentData is the properties applied to an interval of a line.
type lineSegmentData struct {
	ival  interval
	props properties
}

// interval is a closed-open interval of byte indices into a line.
type interval struct {
	beg, end int
}

// lineSegmentPool is a non-thread-safe pool of line segments that reduces
// garbage collection pressure. A line may have many segments.
type lineSegmentPool struct {
	arr []*lineSegment
}

var gLineSegmentPool lineSegmentPool

// newLineSegment returns a pooled line segment.
func newLineSegment() *lineSegment {
	return gLineSegmentPool.get()
}

// releaseLineSegment returns a line segment to the pool.
func releaseLineSegment(s *lineSegment) {
	gLineSegmentPool.put(s)
}

var gTreeNodePool = avltree.WithSyncPool[int, *lineSegment]()

// newLine returns an empty line.
func newLine() *line {
	l := &line{
		segmentIndex: avltree.New(math.CompareOrdered[int], gTreeNodePool),
	}
	l.segmentList.InitLinks()
	return l
}

// init initializes the line with text and a single segment covering it.
// The caller must ensure that the text is not modified afterwards.
func (l *line) init(text []byte) {
	l.text = text
	for _, s := range l.segmentIndex.All() {
		releaseLineSegment(s)
	}
	l.segmentIndex.Clear()
	l.segmentList.InitLinks()

	// Insert a root segment representing the whole line without any
	// properties set. This makes it easier for the text encoder since there
	// won't be any holes in the data. The drawback is that it will be more
	// expensive to generate the line segment properties as more segments
	// have to be split.
	s := newLineSegment()
	s.Value.ival.end = len(text)
	l.insertSegment(s, &l.segmentList)
}

// applyProgram applies the filters of prog according to its apply statements.
// Filter state is reset after every line, also when a condition fails, so that
// a later line can not observe cached match results.
func (l *line) applyProgram(prog *program) error {
	defer prog.globalFilterState.clear()

	for _, stm := range prog.stms {
		doApply, err := stm.cond.Evaluate()
		if err != nil {
			return decorateErrorWithSource(err, prog.name)
		} else if !doApply {
			continue
		}
		stm.filters.apply(l.applyFilter)
	}
	return nil
}

// applyFilter matches f against the line, splices the properties of matched
// groups into the line and applies nested filters.
func (l *line) applyFilter(f *filter) {
	var r [][]int
	if f.regexp != nil {
		r = f.state.match(l.text, f.regexp, true)
	} else if f.regexpFrom != nil {
		r = f.regexpFrom.state.match(l.text, f.regexpFrom.regexp, false)
	}

	applyToRegexpResult(r, func(group int, ival interval) {
		// Only intervals that cover at least one byte can be colored.
		// Intervals of groups that did not match and zero-length matches
		// have nothing to color.
		if ival.len() > 0 {
			if props, ok := f.props[group]; ok {
				l.spliceProperties(ival, props)
			}
		}
	})

	// Apply sub filters
	f.filters.apply(l.applyFilter)
}

// insertSegment inserts newSegment into the index and the linked list after
// prevSegment.
func (l *line) insertSegment(newSegment, prevSegment *lineSegment) {
	l.segmentIndex.Add(newSegment.Value.ival.beg, newSegment)
	prevSegment.LinkNext(newSegment)
}

// spliceProperties merges the properties into every line segment overlapping
// the interval, splitting segments at the interval boundaries as needed.
func (l *line) spliceProperties(ival interval, props properties) {
	// A zero-length interval has no characters to apply properties to.
	if ival.len() <= 0 {
		return
	}

	_, head, found := l.segmentIndex.FindEqualOrLesser(ival.beg)

	// There should always be a line segment in the tree that matches the
	// less or equal search condition with the input interval because the
	// tree is initially seeded with a segment of the whole input line.
	assert(found)

	// For the same reason, the found segment should always overlap with the
	// input interval.
	assert(head.Value.ival.overlapsWith(ival))

	// The starting tree line segment may not align perfectly with the input
	// interval to splice. Possibly split the head so that its start aligns
	// with the input interval.
	if head.Value.ival.beg < ival.beg {
		tail := newLineSegment()
		tail.Value.ival.beg, tail.Value.ival.end, tail.Value.props =
			ival.beg, head.Value.ival.end, head.Value.props
		head.Value.ival.end = tail.Value.ival.beg
		l.insertSegment(tail, head)
		head = tail
	}

	for {
		if head.Value.ival.end <= ival.end {
			head.Value.props.mergeWith(props)
			ival.beg = head.Value.ival.end
			if ival.len() == 0 {
				break
			}
			head = head.Next()
			// The input interval should always overlap with what's already in
			// the tree.
			assert(ival.beg == head.Value.ival.beg)
		} else {
			tail := newLineSegment()
			tail.Value.ival.beg, tail.Value.ival.end, tail.Value.props =
				ival.end, head.Value.ival.end, head.Value.props
			head.Value.ival.end = tail.Value.ival.beg
			head.Value.props.mergeWith(props)
			l.insertSegment(tail, head)
			break
		}
	}
}

// bytesNewline is shared to avoid a byte slice allocation for every line.
var bytesNewline = []byte("\n")

// output writes each line segment through the encoder followed by a newline.
func (l *line) output(w io.Writer, encoder textEncoder) error {
	var err error

	for s := l.segmentList.Next(); s != &l.segmentList; s = s.Next() {
		if encoder, err = encoder(w, s.Value.props, l.text[s.Value.ival.beg:s.Value.ival.end]); err != nil {
			return err
		}
	}
	if _, err = encoder(w, properties{}, bytesNewline); err != nil {
		return err
	}
	return nil
}

// get returns a line segment from the pool or a new one if the pool is empty.
func (pool *lineSegmentPool) get() *lineSegment {
	if n := len(pool.arr); n > 0 {
		s := pool.arr[n-1]
		pool.arr = pool.arr[:n-1]
		return s
	}
	return list.New[lineSegmentData]()
}

// put returns a line segment to the pool after clearing its state.
func (pool *lineSegmentPool) put(s *lineSegment) {
	s.InitLinks()
	s.Value = lineSegmentData{}
	pool.arr = append(pool.arr, s)
}

// len returns the number of bytes covered by the interval.
func (ival interval) len() int {
	return ival.end - ival.beg
}

// overlapsWith reports whether the interval overlaps other.
func (ival interval) overlapsWith(other interval) bool {
	return ival.beg < other.end && ival.end > other.beg
}
