package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

// failAfterWriter returns err once the configured number of writes have
// succeeded. It is used to exercise write error handling.
type failAfterWriter struct {
	remaining int
	err       error
}

// Write succeeds until the configured number of writes have been performed and
// then reports the configured error.
func (w *failAfterWriter) Write(p []byte) (int, error) {
	if w.remaining <= 0 {
		return 0, w.err
	}
	w.remaining--
	return len(p), nil
}

// TestTextEncoderDummy verifies that the dummy encoder passes text through
// unchanged.
func TestTextEncoderDummy(t *testing.T) {
	var buf bytes.Buffer

	encoder, err := textEncoderDummy(&buf, newProperties(colorRed, colorNone), []byte("one"))
	if err != nil {
		t.Fatalf("textEncoderDummy: %v", err)
	}
	if _, err = encoder(&buf, properties{}, []byte(" two")); err != nil {
		t.Fatalf("textEncoderDummy: %v", err)
	}

	if got, want := buf.String(), "one two"; got != want {
		t.Errorf("output = %q, want %q", got, want)
	}
}

// TestTextEncoderANSI verifies the escape codes emitted for each property.
func TestTextEncoderANSI(t *testing.T) {
	tests := []struct {
		name  string
		props properties
		want  string
	}{
		{"no properties", properties{}, "text"},
		{"color", newProperties(colorRed, colorNone), "\x1b[31mtext\x1b[0m"},
		{"background color", newProperties(colorNone, colorBlue), "\x1b[44mtext\x1b[0m"},
		{"modifier", newProperties(colorNone, colorNone, modifierBold), "\x1b[1mtext\x1b[0m"},
		{"all modifiers", newProperties(colorNone, colorNone,
			modifierBold, modifierUnderline, modifierReverse, modifierBlink), "\x1b[1;4;7;5mtext\x1b[0m"},
		{"all properties", newProperties(colorWhite, colorRed, modifierBold, modifierUnderline),
			"\x1b[1;4;37;41mtext\x1b[0m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			if _, err := textEncoderANSI(&buf, tt.props, []byte("text")); err != nil {
				t.Fatalf("textEncoderANSI: %v", err)
			}
			if got := buf.String(); got != tt.want {
				t.Errorf("output = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestTextEncoderANSIShortWrite verifies that a write failure is reported both
// when writing escape codes and when writing the text.
func TestTextEncoderANSIShortWrite(t *testing.T) {
	errBoom := errors.New("write failed")

	tests := []struct {
		name   string
		writer *failAfterWriter
	}{
		// Fails on the very first write, while emitting escape codes.
		{"escape code write", &failAfterWriter{err: errBoom}},
		// Allows the escape codes to be written and fails when writing the text.
		{"text write", &failAfterWriter{remaining: 3, err: errBoom}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := textEncoderANSI(tt.writer, newProperties(colorRed, colorNone), []byte("x")); !errors.Is(err, errBoom) {
				t.Errorf("error = %v, want %v", err, errBoom)
			}
		})
	}
}

// TestTextEncoderTestWriteError verifies that a write failure is reported by
// the test encoder.
func TestTextEncoderTestWriteError(t *testing.T) {
	errBoom := errors.New("write failed")
	if _, err := textEncoderTest(&failAfterWriter{err: errBoom}, properties{}, []byte("x")); !errors.Is(err, errBoom) {
		t.Errorf("write error = %v, want %v", err, errBoom)
	}
}

// TestTextEncoderTestModifierList verifies that multiple modifiers are emitted
// separated by commas.
func TestTextEncoderTestModifierList(t *testing.T) {
	var buf bytes.Buffer
	props := newProperties(colorNone, colorNone, modifierBold, modifierUnderline)
	if _, err := textEncoderTest(&buf, props, []byte("x")); err != nil {
		t.Fatalf("textEncoderTest: %v", err)
	}
	if want := "mod:[bold,underline]"; !strings.Contains(buf.String(), want) {
		t.Errorf("output = %q, want it to contain %q", buf.String(), want)
	}
}

// TestLineOutputError verifies that output reports errors from the encoder.
func TestLineOutputError(t *testing.T) {
	errBoom := errors.New("write failed")

	// A line that is split into two segments fails while encoding the second
	// segment.
	l := newLine()
	l.init([]byte("ab"))
	l.spliceProperties(interval{0, 1}, newProperties(colorRed, colorNone))
	if err := l.output(&failAfterWriter{remaining: 1, err: errBoom}, textEncoderTest); !errors.Is(err, errBoom) {
		t.Errorf("segment output error = %v, want %v", err, errBoom)
	}

	// A line with a single segment fails while encoding the trailing newline.
	single := newLine()
	single.init([]byte("a"))
	if err := single.output(&failAfterWriter{remaining: 1, err: errBoom}, textEncoderTest); !errors.Is(err, errBoom) {
		t.Errorf("newline output error = %v, want %v", err, errBoom)
	}
}
