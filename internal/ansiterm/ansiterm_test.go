package ansiterm

import (
	"bytes"
	"errors"
	"testing"
)

// TestWriteCodes verifies the ANSI escape sequence written for the given
// terminal codes.
func TestWriteCodes(t *testing.T) {
	tests := []struct {
		name  string
		codes []Code
		want  string
	}{
		{"no codes", nil, "\x1b[m"},
		{"reset", []Code{CodeReset}, "\x1b[0m"},
		{"foreground color", []Code{CodeFGRed}, "\x1b[31m"},
		{"intense background color", []Code{CodeBGIRed}, "\x1b[101m"},
		{"multiple codes", []Code{CodeBold, CodeFGRed, CodeBGBlue}, "\x1b[1;31;44m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := WriteCodes(&buf, tt.codes...); err != nil {
				t.Fatalf("WriteCodes: %v", err)
			}
			if got := buf.String(); got != tt.want {
				t.Errorf("output = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestWriteCodesWriteError verifies that a write failure is reported.
func TestWriteCodesWriteError(t *testing.T) {
	errBoom := errors.New("write failed")
	if err := WriteCodes(failingWriter{errBoom}); !errors.Is(err, errBoom) {
		t.Errorf("error = %v, want %v", err, errBoom)
	}
}

// failingWriter reports the configured error on every write.
type failingWriter struct {
	err error
}

// Write reports the configured error.
func (w failingWriter) Write(p []byte) (int, error) {
	return 0, w.err
}
