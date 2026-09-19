package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCreateProgramErrors verifies that invalid configurations are rejected
// with a descriptive error. The cases are grouped by the configuration
// parameter being validated.
func TestCreateProgramErrors(t *testing.T) {
	tests := []struct {
		name    string
		config  string
		wantErr string
	}{
		// Program level parameters.
		{"empty input", ``, "expected one association list"},
		{"malformed syntax", `{`, "unterminated association list"},
		{"trailing data", `{} {}`, "trailing data"},
		{"root not association list", `[]`, "expected association list"},
		{"unknown root parameter", `{bogus: 1}`, "unknown parameter"},
		{"missing filter", `{}`, `missing parameter "filter"`},
		{"missing apply", `{filter: {name: f regexp: (x)}}`, `missing parameter "apply"`},

		// Filter parameters.
		{"filter not association list", `{filter: []}`, "expected association list"},
		{"duplicate filter parameter", `{filter: {name: a name: b regexp: (x)}}`, "duplicate parameter"},
		{"regexp and regexpFrom exclusive", `{filter: {name: f regexp: (x) regexpFrom: f}}`, "mutually exclusive"},
		{"name not string", `{filter: {name: [] regexp: (x)}}`, "expected string"},
		{"name contains separator", `{filter: {name: a/b regexp: (x)}}`, "must not contain"},
		{"regexp not string", `{filter: {name: f regexp: []}}`, "expected string"},
		{"regexp compile error", `{filter: {name: f regexp: "("}}`, "error parsing regexp"},
		{"regexpFrom not string", `{filter: {name: base regexp: (x)} filter: {name: d regexpFrom: []}}`, "expected string"},
		{"regexpFrom missing filter", `{filter: {name: f regexpFrom: missing}}`, "does not exist"},
		{"regexpFrom without regexp", `{filter: {name: base} filter: {name: d regexpFrom: base}}`, "miss regexp"},
		{"duplicate nested filter", `{filter: {name: p filter: {name: c regexp: (a)} filter: {name: c regexp: (b)}}}`, "duplicate filter"},
		{"nested filter parse error", `{filter: {name: p filter: {name: c regexp: []}}}`, "expected string"},
		{"unknown filter parameter", `{filter: {name: f regexp: (x) bogus: 1}}`, "unknown parameter"},
		{"duplicate top level filter", `{filter: {name: f regexp: (a)} filter: {name: f regexp: (b)}}`, "duplicate filter"},

		// Filter properties parameters.
		{"duplicate properties parameter", `{filter: {name: f regexp: (x) properties: { 1: {color: red} } properties: { 1: {bgcolor: blue} }}}`, "duplicate parameter"},
		{"properties not association list", `{filter: {name: f regexp: (x) properties: []}}`, "expected association list"},
		{"property group zero", `{filter: {name: f regexp: (x) properties: { 0: {color: red} }}}`, "invalid regexp group"},
		{"property group not numeric", `{filter: {name: f regexp: (x) properties: { x: {color: red} }}}`, "invalid regexp group"},
		{"duplicate property group", `{filter: {name: f regexp: (x) properties: { 1: {color: red} 1: {color: blue} }}}`, "duplicate regexp group"},
		{"property not association list", `{filter: {name: f regexp: (x) properties: { 1: [] }}}`, "expected association list"},
		{"duplicate property", `{filter: {name: f regexp: (x) properties: { 1: {color: red color: blue} }}}`, "duplicate parameter"},
		{"unknown property", `{filter: {name: f regexp: (x) properties: { 1: {bogus: red} }}}`, "unknown parameter"},
		{"color not string", `{filter: {name: f regexp: (x) properties: { 1: {color: []} }}}`, "expected string"},
		{"unknown color", `{filter: {name: f regexp: (x) properties: { 1: {color: bogus} }}}`, "unknown color"},
		{"unknown bgcolor", `{filter: {name: f regexp: (x) properties: { 1: {bgcolor: bogus} }}}`, "unknown color"},
		{"modifiers not list", `{filter: {name: f regexp: (x) properties: { 1: {modifiers: {}} }}}`, "expected list"},
		{"modifier element not string", `{filter: {name: f regexp: (x) properties: { 1: {modifiers: [[]]} }}}`, "expected string"},
		{"unknown modifier", `{filter: {name: f regexp: (x) properties: { 1: {modifiers: bogus} }}}`, "unknown modifier"},
		{"properties without regexp", `{filter: {name: f properties: { 1: {color: red} }}}`, "properties set but filter has no regexp"},
		{"property group out of range", `{filter: {name: base regexp: (x)} filter: {name: d regexpFrom: base properties: { 2: {color: red} }}}`, "invalid regexp group"},

		// Apply parameters.
		{"apply not association list", `{filter: {name: f regexp: (x)} apply: []}`, "expected association list"},
		{"duplicate apply parameter", `{filter: {name: f regexp: (x)} apply: {filters: f filters: f}}`, "duplicate parameter"},
		{"unknown apply parameter", `{filter: {name: f regexp: (x)} apply: {filters: f bogus: 1}}`, "unknown parameter"},
		{"apply missing filter", `{filter: {name: f regexp: (x)} apply: {filters: missing}}`, "does not exist"},
		{"apply missing filters", `{filter: {name: f regexp: (x)} apply: {cond: [not [filter-match? f]]}}`, `missing parameter "filters"`},
		{"apply filters not list", `{filter: {name: f regexp: (x)} apply: {filters: {}}}`, "expected list"},
		{"apply filter element not string", `{filter: {name: f regexp: (x)} apply: {filters: [[]]}}`, "expected string"},

		// Condition expressions.
		{"condition not list", `{filter: {name: f regexp: (x)} apply: {cond: (x) filters: f}}`, "expected list"},
		{"condition missing function", `{filter: {name: f regexp: (x)} apply: {cond: [] filters: f}}`, "missing function name"},
		{"condition function not string", `{filter: {name: f regexp: (x)} apply: {cond: [{}] filters: f}}`, "expected function name"},
		{"condition unknown function", `{filter: {name: f regexp: (x)} apply: {cond: [bogus] filters: f}}`, "unknown function"},
		{"condition invalid argument", `{filter: {name: f regexp: (x)} apply: {cond: [not {}] filters: f}}`, "expected string or function call"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := createProgram(strings.NewReader(tt.config)); err == nil {
				t.Fatal("expected an error")
			} else if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want it to contain %q", err, tt.wantErr)
			}
		})
	}
}

// TestLoadProgramErrors verifies that loadProgram reports a configuration file
// that can not be opened or parsed.
func TestLoadProgramErrors(t *testing.T) {
	t.Run("missing file", func(t *testing.T) {
		if _, err := loadProgram("no-such-file.rainbow"); err == nil {
			t.Fatal("expected an error")
		}
	})

	t.Run("invalid configuration", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "invalid.rainbow")
		if err := os.WriteFile(path, []byte("{"), 0o600); err != nil {
			t.Fatalf("write configuration file: %v", err)
		}

		_, err := loadProgram(path)
		if err == nil {
			t.Fatal("expected an error")
		}
		if !strings.Contains(err.Error(), path) {
			t.Errorf("error = %q, want it to contain the source %q", err, path)
		}
	})
}

// TestDecorateErrorWithSource verifies that the source is prepended to an error
// and that a space is inserted unless the error already starts with a position.
func TestDecorateErrorWithSource(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{"text error", errors.New("boom"), "config: boom"},
		{"positional error", errors.New("1:2: boom"), "config:1:2: boom"},
		{"empty error", errors.New(""), "config:"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := decorateErrorWithSource(tt.err, "config").Error(); got != tt.want {
				t.Errorf("error = %q, want %q", got, tt.want)
			}
		})
	}
}
