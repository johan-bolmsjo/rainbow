package main

import (
	"fmt"
)

type color uint8

const (
	colorNone color = iota
	colorBlack
	colorRed
	colorGreen
	colorYellow
	colorBlue
	colorMagenta
	colorCyan
	colorWhite
	colorIBlack
	colorIRed
	colorIGreen
	colorIYellow
	colorIBlue
	colorIMagenta
	colorICyan
	colorIWhite
)

var itoaColor = map[color]string{
	colorNone:     "none",
	colorBlack:    "black",
	colorRed:      "red",
	colorGreen:    "green",
	colorYellow:   "yellow",
	colorBlue:     "blue",
	colorMagenta:  "magenta",
	colorCyan:     "cyan",
	colorWhite:    "white",
	colorIBlack:   "iblack",
	colorIRed:     "ired",
	colorIGreen:   "igreen",
	colorIYellow:  "iyellow",
	colorIBlue:    "iblue",
	colorIMagenta: "imagenta",
	colorICyan:    "icyan",
	colorIWhite:   "iwhite",
}

var atoiColor = func() map[string]color {
	m := map[string]color{}
	for k, v := range itoaColor {
		m[v] = k
	}
	return m
}()

func parseColor(s string) (color, error) {
	if c, ok := atoiColor[s]; ok {
		return c, nil
	}
	return colorNone, fmt.Errorf("unknown color %q", s)
}

func (c color) String() string {
	return itoaColor[c]
}
