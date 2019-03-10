package main

import (
	"github.com/johan-bolmsjo/gods/avltree"
	"github.com/johan-bolmsjo/gods/list"
	"io"
)

type line struct {
	line        []byte        // shared: must not be modified
	segmentTree *avltree.Tree // Tree of *lineSegment
	segmentList list.Elem     // List of *lineSegment
}

type lineSegment struct {
	elem  list.Elem // Linked list of segments in ascending order
	ival  interval
	props properties
}

// non-thread safe line segment pool to ease GC pressure.
// lots of these objects may be created per line.
type lineSegmentPool struct {
	elems []*lineSegment
}

// Closed open interval (byte indices) of line slice
type interval struct {
	beg, end int
}

var gLineSegmentPool lineSegmentPool

func newLine() *line {
	l := new(line)
	l.segmentTree = avltree.New(
		func(d avltree.Data) avltree.Key { return &d.(*lineSegment).ival.beg },
		func(lhs, rhs avltree.Key) int { return *lhs.(*int) - *rhs.(*int) },
	)
	l.segmentList.Init(nil)
	return l
}

func (l *line) init(line []byte) {
	l.line = line
	l.segmentTree.Clear(func(d avltree.Data) {
		freeLineSegment(d.(*lineSegment))
	})
	l.segmentList.Init(nil)

	// Insert a root segment representing the whole line without any properties set. This makes
	// it easier for the control codes encoder since there wont be any holes in the data. The
	// drawback is that it will be more expensive to generate the line segment properties as
	// more segments have to be split.
	s := newLineSegment()
	s.ival.end = len(line)
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
		r = f.state.match(l.line, f.regexp, true)
	} else if f.regexpFrom != nil {
		r = f.regexpFrom.state.match(l.line, f.regexpFrom.regexp, false)
	}

	applyToRegexpResult(r, func(group int, ival interval) {
		if props, ok := f.props[group]; ok {
			l.spliceProperties(ival, props)
		}
	})

	// Apply sub filters
	f.filters.apply(l.applyFilter)
}

func (l *line) insertSegment(s *lineSegment, nextTo *list.Elem) {
	l.segmentTree.Insert(s)
	nextTo.LinkNext(&s.elem)
}

// Splice line properties with line segments in tree.
func (l *line) spliceProperties(ival interval, props properties) {
	head := l.segmentTree.FindLe(&ival.beg).(*lineSegment)

	// There should always be a line segment in the tree that matches the
	// less or equal search condition with the input interval because the tree
	// is initially seeded with a segment of the whole input line.
	assert(head != nil)

	// For the same reason the found segment should always overlap with the input interval.
	assert(head.ival.overlapsWith(ival))

	// The starting tree line segment may not align perfectly with the input interval to splice.
	// Possibly split the head so that its start aligns with the input interval.
	if head.ival.beg < ival.beg {
		tail := newLineSegment()
		tail.ival.beg, tail.ival.end, tail.props = ival.beg, head.ival.end, head.props
		head.ival.end = tail.ival.beg

		head.elem.LinkNext(&tail.elem)
		l.insertSegment(tail, &head.elem)
		head = tail
	}

	for {
		if head.ival.end <= ival.end {
			head.props.mergeWith(props)

			ival.beg = head.ival.end
			if ival.len() == 0 {
				break
			}

			head = head.elem.Next().Value.(*lineSegment)
			// The input interval should always overlap with what's already in the tree.
			assert(ival.beg == head.ival.beg)
		} else {
			tail := newLineSegment()
			tail.ival.beg, tail.ival.end, tail.props = ival.end, head.ival.end, head.props
			head.ival.end = tail.ival.beg

			head.props.mergeWith(props)

			l.insertSegment(tail, &head.elem)
			break
		}
	}
}

// Compiler does not seem smart enough to avoid memory allocations when directly
// passing []byte("...") to a function accepting a byte slice.
var bytesNewline = []byte("\n")

func (l *line) output(w io.Writer, encoder textEncoder) error {
	var err error

	for e := l.segmentList.Next(); e != &l.segmentList; e = e.Next() {
		s := e.Value.(*lineSegment)
		if encoder, err = encoder(w, s.props, l.line[s.ival.beg:s.ival.end]); err != nil {
			return err
		}
	}
	if _, err = encoder(w, properties{}, bytesNewline); err != nil {
		return err
	}
	return nil
}

func newLineSegment() *lineSegment {
	return gLineSegmentPool.get()
}

func freeLineSegment(s *lineSegment) {
	gLineSegmentPool.put(s)
}

func (s *lineSegment) clear() {
	*s = lineSegment{}
	s.elem.Init(s)
}

func (pool *lineSegmentPool) get() *lineSegment {
	if n := len(pool.elems); n > 0 {
		s := pool.elems[n-1]
		pool.elems = pool.elems[:n-1]
		return s
	}
	s := new(lineSegment)
	s.elem.Init(s)
	return s
}

func (pool *lineSegmentPool) put(s *lineSegment) {
	s.clear()
	pool.elems = append(pool.elems, s)
}

func (ival interval) len() int {
	return ival.end - ival.beg
}

func (ival interval) overlapsWith(other interval) bool {
	return ival.beg < other.end && ival.end > other.beg
}
