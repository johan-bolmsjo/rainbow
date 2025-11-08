package main

import (
	"github.com/johan-bolmsjo/gods/v4/avltree"
	"github.com/johan-bolmsjo/gods/v4/list"
	"github.com/johan-bolmsjo/gods/v4/math"
	"io"
)

type line struct {
	text         []byte // shared data, must not be modified after initialization
	segmentIndex *avltree.Tree[int, *lineSegment]
	segmentList  lineSegment
}

// Linked list of segments in ascending order
type lineSegment = list.Node[lineSegmentData]

type lineSegmentData struct {
	ival  interval
	props properties
}

// Closed open interval (byte indices) of line slice
type interval struct {
	beg, end int
}

// Non-thread safe line segment pool to ease GC pressure.
// A line may have many segments.
type lineSegmentPool struct {
	arr []*lineSegment
}

var gLineSegmentPool lineSegmentPool

func newLineSegment() *lineSegment {
	return gLineSegmentPool.get()
}

func releaseLineSegment(s *lineSegment) {
	gLineSegmentPool.put(s)
}

var gTreeNodePool = avltree.WithSyncPool[int, *lineSegment]()

func newLine() *line {
	l := &line{
		segmentIndex: avltree.New(math.CompareOrdered[int], gTreeNodePool),
	}
	l.segmentList.InitLinks()
	return l
}

func (l *line) init(text []byte) {
	l.text = text
	for _, s := range l.segmentIndex.All() {
		releaseLineSegment(s)
	}
	l.segmentIndex.Clear()
	l.segmentList.InitLinks()

	// Insert a root segment representing the whole line without any
	// properties set. This makes it easier for the control codes encoder
	// since there wont be any holes in the data. The drawback is that it
	// will be more expensive to generate the line segment properties as
	// more segments have to be split.
	s := newLineSegment()
	s.Value.ival.end = len(text)
	l.insertSegment(s, &l.segmentList)
}

func (l *line) applyProgram(prog *program) error {
	for _, stm := range prog.stms {
		doApply, err := stm.cond.Eval()
		if err != nil {
			return decorateErrorWithSource(err, prog.name)
		} else if !doApply {
			continue
		}
		stm.filters.apply(l.applyFilter)
	}

	prog.globalFilterState.clear()
	return nil
}

func (l *line) applyFilter(f *filter) {
	var r [][]int
	if f.regexp != nil {
		r = f.state.match(l.text, f.regexp, true)
	} else if f.regexpFrom != nil {
		r = f.regexpFrom.state.match(l.text, f.regexpFrom.regexp, false)
	}

	applyToRegexpResult(r, func(group int, ival interval) {
		if ival.beg != -1 {
			if props, ok := f.props[group]; ok {
				l.spliceProperties(ival, props)
			}
		}
	})

	// Apply sub filters
	f.filters.apply(l.applyFilter)
}

func (l *line) insertSegment(newSegment, prevSegment *lineSegment) {
	l.segmentIndex.Add(newSegment.Value.ival.beg, newSegment)
	prevSegment.LinkNext(newSegment)
}

// Splice line properties with line segments in tree.
func (l *line) spliceProperties(ival interval, props properties) {
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

// The compiler does not seem smart enough to avoid memory allocations when directly
// passing []byte("...") to a function accepting a byte slice.
var bytesNewline = []byte("\n")

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

func (pool *lineSegmentPool) get() *lineSegment {
	if n := len(pool.arr); n > 0 {
		s := pool.arr[n-1]
		pool.arr = pool.arr[:n-1]
		return s
	}
	return list.New[lineSegmentData]()
}

func (pool *lineSegmentPool) put(s *lineSegment) {
	s.InitLinks()
	s.Value = lineSegmentData{}
	pool.arr = append(pool.arr, s)
}

func (ival interval) len() int {
	return ival.end - ival.beg
}

func (ival interval) overlapsWith(other interval) bool {
	return ival.beg < other.end && ival.end > other.beg
}
