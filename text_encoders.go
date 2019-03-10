package main

import (
	"bytes"
	"github.com/johan-bolmsjo/errors"
	"github.com/johan-bolmsjo/rainbow/internal/ansiterm"
	"io"
)

// textEncoder writes escape codes to w according to props.
// The function is used to start encoding escape codes for a line. It may return
// a new function which will be used used to encode the next set of properties.
// This way it's possible to implement delta encoding schemes.
type textEncoder func(w io.Writer, props properties, text []byte) (textEncoder, error)

// textEncoderDummy does not emit any escape codes.
func textEncoderDummy(w io.Writer, props properties, text []byte) (textEncoder, error) {
	_, err := w.Write(text)
	return textEncoderDummy, err
}

var foregroundANSIColorCode = map[color]ansiterm.Code{
	colorBlack:    ansiterm.CodeFGBlack,
	colorRed:      ansiterm.CodeFGRed,
	colorGreen:    ansiterm.CodeFGGreen,
	colorYellow:   ansiterm.CodeFGYellow,
	colorBlue:     ansiterm.CodeFGBlue,
	colorMagenta:  ansiterm.CodeFGMagenta,
	colorCyan:     ansiterm.CodeFGCyan,
	colorWhite:    ansiterm.CodeFGWhite,
	colorIBlack:   ansiterm.CodeFGIBlack,
	colorIRed:     ansiterm.CodeFGIRed,
	colorIGreen:   ansiterm.CodeFGIGreen,
	colorIYellow:  ansiterm.CodeFGIYellow,
	colorIBlue:    ansiterm.CodeFGIBlue,
	colorIMagenta: ansiterm.CodeFGIMagenta,
	colorICyan:    ansiterm.CodeFGICyan,
	colorIWhite:   ansiterm.CodeFGIWhite,
}

var backgroundANSIColorCode = map[color]ansiterm.Code{
	colorBlack:    ansiterm.CodeBGBlack,
	colorRed:      ansiterm.CodeBGRed,
	colorGreen:    ansiterm.CodeBGGreen,
	colorYellow:   ansiterm.CodeBGYellow,
	colorBlue:     ansiterm.CodeBGBlue,
	colorMagenta:  ansiterm.CodeBGMagenta,
	colorCyan:     ansiterm.CodeBGCyan,
	colorWhite:    ansiterm.CodeBGWhite,
	colorIBlack:   ansiterm.CodeBGIBlack,
	colorIRed:     ansiterm.CodeBGIRed,
	colorIGreen:   ansiterm.CodeBGIGreen,
	colorIYellow:  ansiterm.CodeBGIYellow,
	colorIBlue:    ansiterm.CodeBGIBlue,
	colorIMagenta: ansiterm.CodeBGIMagenta,
	colorICyan:    ansiterm.CodeBGICyan,
	colorIWhite:   ansiterm.CodeBGIWhite,
}

// textEncoderANSI emits ANSI terminal escape codes.
// See https://en.wikipedia.org/wiki/ANSI_escape_code#Colors
//
// NOTE: Delta encoding of escape codes is currently not performed. Probably not
//       worth the complexity to save a couple of bytes of output.
func textEncoderANSI(w io.Writer, props properties, text []byte) (textEncoder, error) {
	var es errors.Sink

	doWriteCodes := props != properties{}

	if doWriteCodes {
		var codeBuf [10]ansiterm.Code
		codes := codeBuf[:0]

		props.modifiers.foreach(func(m modifier) {
			code := ansiterm.CodeReset
			switch m {
			case modifierBold:
				code = ansiterm.CodeBold
			case modifierUnderline:
				code = ansiterm.CodeUnderline
			case modifierReverse:
				code = ansiterm.CodeReverseVideo
			case modifierBlink:
				code = ansiterm.CodeSlowBlink
			}
			if code != ansiterm.CodeReset {
				codes = append(codes, code)
			}
		})

		if color := props.fgcolor; color != colorNone {
			codes = append(codes, foregroundANSIColorCode[color])
		}
		if color := props.bgcolor; color != colorNone {
			codes = append(codes, backgroundANSIColorCode[color])
		}
		es.Send(ansiterm.WriteCodes(w, codes...))
	}

	if es.Ok() {
		_, err := w.Write(text)
		es.Send(err)
	}
	if doWriteCodes && es.Ok() {
		es.Send(ansiterm.WriteCodes(w, ansiterm.CodeReset))
	}

	return textEncoderANSI, es.Cause()
}

// textEncoderTest emits escape codes used for testing purpose.
func textEncoderTest(w io.Writer, props properties, text []byte) (textEncoder, error) {
	var bb bytes.Buffer

	bb.WriteString("fg:")
	bb.WriteString(props.fgcolor.String())
	bb.WriteString(",bg:")
	bb.WriteString(props.bgcolor.String())

	bb.WriteString(",mod:[")
	emittedModifier := false
	props.modifiers.foreach(func(m modifier) {
		if emittedModifier {
			bb.WriteString(",")
		}
		bb.WriteString(m.String())
		emittedModifier = true
	})
	bb.WriteString("]")

	if pad := 40 - bb.Len(); pad > 0 {
		bb.Write(bytes.Repeat([]byte(" "), pad))
	}

	bb.WriteString("{")
	bb.Write(text)
	bb.WriteString("}")
	bb.WriteString("\n")

	_, err := w.Write(bb.Bytes())
	return textEncoderTest, err
}
