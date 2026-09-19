package main

// applyToRegexpResult applies f to each capturing group of every match in the
// regexp result. Group numbers count from one for the leftmost capturing
// parenthesis.
func applyToRegexpResult(res [][]int, f func(group int, ival interval)) {
	const (
		// resultPairSize is the number of result values per match or capturing group.
		resultPairSize = 2

		// wholeMatchPairCount is the number of leading result pairs that describe the
		// whole match rather than a capturing group.
		wholeMatchPairCount = 1
	)
	firstGroup := wholeMatchPairCount * resultPairSize
	for _, a := range res {
		for i := firstGroup; i < len(a); i += resultPairSize {
			f(i/resultPairSize, interval{a[i], a[i+1]})
		}
	}
}
