package ansiterm

import (
	"github.com/johan-bolmsjo/errors"
	"io"
	"strconv"
)

// ANSI terminal escape code.
// See https://en.wikipedia.org/wiki/ANSI_escape_code
type Code uint8

const escapeSeq = "\x1b["

var (
	escapeSeqBegin = []byte(escapeSeq)
	escapeSeqEnd   = []byte("m")
)

const (
	CodeReset Code = iota
	CodeBold
	CodeFaint
	CodeItalic
	CodeUnderline
	CodeSlowBlink
	CodeRapidBlink
	CodeReverseVideo
	CodeConceal
	CodeCrossedOut
)

// Foreground colors.
const (
	CodeFGBlack Code = iota + 30
	CodeFGRed
	CodeFGGreen
	CodeFGYellow
	CodeFGBlue
	CodeFGMagenta
	CodeFGCyan
	CodeFGWhite
)

// Background colors.
const (
	CodeBGBlack Code = iota + 40
	CodeBGRed
	CodeBGGreen
	CodeBGYellow
	CodeBGBlue
	CodeBGMagenta
	CodeBGCyan
	CodeBGWhite
)

// Intense foreground colors.
const (
	CodeFGIBlack Code = iota + 90
	CodeFGIRed
	CodeFGIGreen
	CodeFGIYellow
	CodeFGIBlue
	CodeFGIMagenta
	CodeFGICyan
	CodeFGIWhite
)

// Intense background colors.
const (
	CodeBGIBlack Code = iota + 100
	CodeBGIRed
	CodeBGIGreen
	CodeBGIYellow
	CodeBGIBlue
	CodeBGIMagenta
	CodeBGICyan
	CodeBGIWhite
)

// Write ANSI terminal codes to w.
func WriteCodes(w io.Writer, codes ...Code) error {
	var codeBuf [4]byte

	var es errors.Sink
	write := func(b []byte) {
		if es.Ok() {
			_, err := w.Write(b)
			es.Send(err)
		}
	}

	write(escapeSeqBegin)

	codeEmitted := false
	for _, code := range codes {
		if codeEmitted {
			write([]byte(";"))
		}
		write(strconv.AppendUint(codeBuf[:0], uint64(code), 10))
		codeEmitted = true
	}

	write(escapeSeqEnd)
	return es.Cause()
}
