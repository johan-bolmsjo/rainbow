package main

// assert panics if the condition is false. It is used to document invariants
// that must hold for the program to behave correctly.
func assert(cond bool) {
	if !cond {
		panic("Highly Illogical...")
	}
}
