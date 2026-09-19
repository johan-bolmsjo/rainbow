package igor

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/johan-bolmsjo/saft"
)

// compileCondition parses and compiles a condition expression from source.
func compileCondition(t *testing.T, source string) *Condition {
	t.Helper()

	elems, err := saft.Parse(strings.NewReader(source))
	if err != nil {
		t.Fatalf("saft.Parse(%q): %v", source, err)
	}
	if len(elems) != 1 {
		t.Fatalf("saft.Parse(%q) returned %d elements, want 1", source, len(elems))
	}

	cond, err := NewInterpreter().CompileCondition(elems[0])
	if err != nil {
		t.Fatalf("CompileCondition(%q): %v", source, err)
	}
	return cond
}

// TestConditionEvaluate verifies evaluation of the built in logical and comparison
// functions, including short circuit behavior.
func TestConditionEvaluate(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   bool
	}{
		{"not true", "[not true]", false},
		{"not nested", "[not [equal? a b]]", true},
		{"and no arguments", "[and]", true},
		{"and last argument", "[and a b]", true},
		{"and short circuits on false", "[and [not true] a]", false},
		{"or no arguments", "[or]", false},
		{"or first argument true", "[or a [not true]]", true},
		{"or second argument true", "[or [not true] a]", true},
		{"equal strings", "[equal? a a]", true},
		{"unequal strings", "[equal? a b]", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := compileCondition(t, tt.source).Evaluate()
			if err != nil {
				t.Fatalf("Evaluate: %v", err)
			}
			if got != tt.want {
				t.Errorf("Evaluate(%q) = %v, want %v", tt.source, got, tt.want)
			}
		})
	}
}

// TestConditionNilEvaluate verifies that a missing condition evaluates to true.
func TestConditionNilEvaluate(t *testing.T) {
	var cond *Condition
	got, err := cond.Evaluate()
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if !got {
		t.Error("Evaluate() = false, want true")
	}
}

// TestCompileConditionErrors verifies that invalid condition expressions are
// rejected when compiled.
func TestCompileConditionErrors(t *testing.T) {
	tests := []struct {
		name    string
		source  string
		wantErr string
	}{
		{"not a list", "x", "expected list"},
		{"missing function name", "[]", "missing function name"},
		{"function name not string", "[{}]", "expected function name"},
		{"unknown function", "[bogus]", "unknown function"},
		{"nested unknown function", "[not [bogus]]", "unknown function"},
		{"invalid argument", "[not {}]", "expected string or function call"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			elems, err := saft.Parse(strings.NewReader(tt.source))
			if err != nil {
				t.Fatalf("saft.Parse: %v", err)
			}
			if _, err = NewInterpreter().CompileCondition(elems[0]); err == nil {
				t.Fatal("expected an error")
			} else if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want it to contain %q", err, tt.wantErr)
			}
		})
	}
}

// TestConditionEvaluateErrors verifies that argument errors are reported when
// evaluating a condition.
func TestConditionEvaluateErrors(t *testing.T) {
	tests := []struct {
		name    string
		source  string
		wantErr string
	}{
		{"not argument count", "[not]", "invalid number of arguments"},
		{"not too many arguments", "[not a b]", "invalid number of arguments"},
		{"equal argument count", "[equal? a]", "invalid number of arguments"},
		{"equal too many arguments", "[equal? a b c]", "invalid number of arguments"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := compileCondition(t, tt.source).Evaluate()
			if err == nil {
				t.Fatal("expected an error")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want it to contain %q", err, tt.wantErr)
			}
		})
	}
}

// TestObjectIsEqual verifies equality between the interpreter object types.
func TestObjectIsEqual(t *testing.T) {
	tests := []struct {
		name string
		lhs  Object
		rhs  Object
		want ObjectBool
	}{
		{"equal bools", ObjectBool(true), ObjectBool(true), true},
		{"unequal bools", ObjectBool(true), ObjectBool(false), false},
		{"bool and string", ObjectBool(true), ObjectString("true"), false},
		{"equal strings", ObjectString("a"), ObjectString("a"), true},
		{"unequal strings", ObjectString("a"), ObjectString("b"), false},
		{"equal string lists", ObjectStringList{"a", "b"}, ObjectStringList{"a", "b"}, true},
		{"unequal string list lengths", ObjectStringList{"a"}, ObjectStringList{"a", "b"}, false},
		{"unequal string list values", ObjectStringList{"a"}, ObjectStringList{"b"}, false},
		{"other types", ObjectNone{}, ObjectNone{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := objectIsEqual(tt.lhs, tt.rhs); got != tt.want {
				t.Errorf("objectIsEqual(%v, %v) = %v, want %v", tt.lhs, tt.rhs, got, tt.want)
			}
		})
	}
}

// TestObjectIsTrue verifies which interpreter objects are considered true.
func TestObjectIsTrue(t *testing.T) {
	tests := []struct {
		name string
		obj  Object
		want ObjectBool
	}{
		{"none", ObjectNone{}, false},
		{"false bool", ObjectBool(false), false},
		{"true bool", ObjectBool(true), true},
		{"empty string", ObjectString(""), true},
		{"nil string list", ObjectStringList(nil), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := objectIsTrue(tt.obj); got != tt.want {
				t.Errorf("objectIsTrue(%v) = %v, want %v", tt.obj, got, tt.want)
			}
		})
	}
}

// TestObjectTypes verifies the reported type of each interpreter object.
func TestObjectTypes(t *testing.T) {
	tests := []struct {
		name string
		obj  Object
		want Type
	}{
		{"none", ObjectNone{}, TypeNone},
		{"bool", ObjectBool(true), TypeBool},
		{"call", &objectCall{}, TypeCall},
		{"string", ObjectString("a"), TypeString},
		{"string list", ObjectStringList{"a"}, TypeStringList},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.obj.Type(); got != tt.want {
				t.Errorf("Type() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestTypeString verifies the textual representation of each interpreter type.
func TestTypeString(t *testing.T) {
	tests := []struct {
		typeValue Type
		want      string
	}{
		{TypeNone, "None"},
		{TypeBool, "Bool"},
		{TypeCall, "Call"},
		{TypeString, "String"},
		{TypeStringList, "StringList"},
		{Type(255), "?"},
	}

	for _, tt := range tests {
		if got := tt.typeValue.String(); got != tt.want {
			t.Errorf("Type(%d).String() = %q, want %q", tt.typeValue, got, tt.want)
		}
	}
}

// TestExceptionErrors verifies the error messages produced by the exception
// helpers.
func TestExceptionErrors(t *testing.T) {
	tests := []struct {
		name string
		got  string
		want string
	}{
		{
			"argument count with expectation",
			ExceptionInvalidNumberOfArguments(1, "2").Error(),
			"invalid number of arguments: 1, expected: 2",
		},
		{
			"argument count without expectation",
			ExceptionInvalidNumberOfArguments(1, "").Error(),
			"invalid number of arguments: 1",
		},
		{
			"invalid argument",
			ExceptionInvalidArgument(0, "missing").Error(),
			"invalid argument: 0, missing",
		},
		{
			"type error",
			ExceptionTypeError(ObjectString("x"), 1, TypeBool, TypeString).Error(),
			"type error: argument 1 (String) is not Bool|String",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("error = %q, want %q", tt.got, tt.want)
			}
		})
	}
}

// TestCatchAndDecorate verifies that a thrown exception is caught and decorated
// using the supplied function.
func TestCatchAndDecorate(t *testing.T) {
	var err error
	func() {
		defer catchAndDecorate(&err, func(err error) error {
			return fmt.Errorf("decorated: %w", err)
		})
		Throw(errors.New("boom"))
	}()

	if err == nil {
		t.Fatal("expected an error")
	}
	if got, want := err.Error(), "decorated: boom"; got != want {
		t.Errorf("error = %q, want %q", got, want)
	}
}

// TestExceptionHandling verifies that the exception helpers decorate thrown
// exceptions and let unrelated panics propagate.
func TestExceptionHandling(t *testing.T) {
	t.Run("catch repanics", func(t *testing.T) {
		defer func() {
			if got := recover(); got != "boom" {
				t.Errorf("recover() = %v, want %q", got, "boom")
			}
		}()
		func() {
			defer catch(new(error))
			panic("boom")
		}()
	})

	t.Run("catchAndDecorate repanics", func(t *testing.T) {
		defer func() {
			if got := recover(); got != "boom" {
				t.Errorf("recover() = %v, want %q", got, "boom")
			}
		}()
		func() {
			defer catchAndDecorate(new(error), func(err error) error { return err })
			panic("boom")
		}()
	})

	t.Run("decorateException decorates", func(t *testing.T) {
		defer func() {
			e, ok := recover().(*exception)
			if !ok {
				t.Fatal("expected an exception panic")
			}
			if got, want := e.err.Error(), "decorated: boom"; got != want {
				t.Errorf("error = %q, want %q", got, want)
			}
		}()
		func() {
			defer decorateException(func(err error) error {
				return fmt.Errorf("decorated: %w", err)
			})
			Throw(errors.New("boom"))
		}()
	})

	t.Run("decorateException repanics", func(t *testing.T) {
		defer func() {
			if got := recover(); got != "boom" {
				t.Errorf("recover() = %v, want %q", got, "boom")
			}
		}()
		func() {
			defer decorateException(func(err error) error { return err })
			panic("boom")
		}()
	})
}
