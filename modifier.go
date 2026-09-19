package main

import (
	"fmt"
)

// modifier identifies a terminal text modifier.
type modifier uint8

const (
	modifierBold modifier = iota
	modifierUnderline
	modifierReverse
	modifierBlink
)

const firstModifier = modifierBold
const lastModifier = modifierBlink

// itoaModifier maps a modifier to its configuration name.
var itoaModifier = map[modifier]string{
	modifierBold:      "bold",
	modifierUnderline: "underline",
	modifierReverse:   "reverse",
	modifierBlink:     "blink",
}

// atoiModifier maps a configuration name to a modifier.
var atoiModifier = func() map[string]modifier {
	m := map[string]modifier{}
	for k, v := range itoaModifier {
		m[v] = k
	}
	return m
}()

// parseModifier returns the modifier with the given configuration name.
func parseModifier(s string) (modifier, error) {
	if m, ok := atoiModifier[s]; ok {
		return m, nil
	}
	return modifierBold, fmt.Errorf("unknown modifier %q", s)
}

// String returns the configuration name of the modifier.
func (m modifier) String() string {
	return itoaModifier[m]
}

// modifierSet is a bit set of modifiers.
type modifierSet uint8

// set adds m to the set.
func (s *modifierSet) set(m modifier) {
	*s |= modifierSet(1 << m)
}

// test reports whether m is a member of the set.
func (s *modifierSet) test(m modifier) bool {
	return *s&modifierSet(1<<m) != 0
}

// foreach calls f for every modifier in the set in ascending order.
func (s *modifierSet) foreach(f func(m modifier)) {
	for m := firstModifier; m <= lastModifier; m++ {
		if s.test(m) {
			f(m)
		}
	}
}
