package main

import (
	"fmt"
)

type modifier uint8

const (
	modifierBold modifier = iota
	modifierUnderline
	modifierReverse
	modifierBlink
)

const firstModifier = modifierBold
const lastModifier = modifierBlink

var itoaModifier = map[modifier]string{
	modifierBold:      "bold",
	modifierUnderline: "underline",
	modifierReverse:   "reverse",
	modifierBlink:     "blink",
}

var atoiModifier = func() map[string]modifier {
	m := map[string]modifier{}
	for k, v := range itoaModifier {
		m[v] = k
	}
	return m
}()

func parseModifier(s string) (modifier, error) {
	if m, ok := atoiModifier[s]; ok {
		return m, nil
	}
	return modifierBold, fmt.Errorf("unknown modifier %q", s)
}

func (m modifier) String() string {
	return itoaModifier[m]
}

type modifierSet uint8

func (s *modifierSet) set(m modifier) {
	*s |= modifierSet(1 << m)
}

func (s *modifierSet) test(m modifier) bool {
	return *s&modifierSet(1<<m) != 0
}

func (s *modifierSet) foreach(f func(m modifier)) {
	for m := firstModifier; m <= lastModifier; m++ {
		if s.test(m) {
			f(m)
		}
	}
}
