package main

// Apply function to go stdlib regexp result.
func applyToRegexpResult(res [][]int, f func(group int, ival interval)) {
	for _, a := range res {
		for i := 2; i < len(a); i += 2 { // Skip the first "whole regexp" match; only include the groups
			f(i/2, interval{a[i], a[i+1]})
		}
	}
}
