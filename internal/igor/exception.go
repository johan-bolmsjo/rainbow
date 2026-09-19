package igor

import (
	"fmt"
	"strings"
)

// exception wraps an error thrown by the interpreter.
type exception struct {
	err error
}

// Throw throws an exception that can be caught by catch or catchAndDecorate.
func Throw(err error) {
	panic(&exception{err})
}

// catch catches any exception thrown when evaluating an expression.
// Example use: defer catch(&err)
func catch(err *error) {
	if x := recover(); x != nil {
		if e, ok := x.(*exception); ok {
			*err = e.err
		} else {
			panic(x)
		}
	}
}

// catchAndDecorate catches an exception like catch and decorates the resulting
// error using the supplied function.
func catchAndDecorate(err *error, decorate func(err error) error) {
	if x := recover(); x != nil {
		if e, ok := x.(*exception); ok {
			*err = decorate(e.err)
		} else {
			panic(x)
		}
	}
}

// decorateException decorates any thrown exception using the supplied function.
// Example: defer decorateException(...)
func decorateException(decorate func(err error) error) {
	if x := recover(); x != nil {
		if e, ok := x.(*exception); ok {
			e.err = decorate(e.err)
			panic(e)
		} else {
			panic(x)
		}
	}
}

// ExceptionInvalidNumberOfArguments is thrown on invalid number of arguments.
func ExceptionInvalidNumberOfArguments(args int, expected string) error {
	if expected == "" {
		return fmt.Errorf("invalid number of arguments: %d", args)
	} else {
		return fmt.Errorf("invalid number of arguments: %d, expected: %s", args, expected)
	}
}

// ExceptionInvalidArgument is thrown on an invalid argument.
func ExceptionInvalidArgument(argNum int, reason string) error {
	return fmt.Errorf("invalid argument: %d, %s", argNum, reason)
}

// ExceptionTypeError is thrown on type errors.
func ExceptionTypeError(arg Object, argNum int, accepted ...Type) error {
	var sb strings.Builder
	for _, t := range accepted {
		if sb.Len() > 0 {
			sb.WriteByte('|')
		}
		sb.WriteString(t.String())
	}
	return fmt.Errorf("type error: argument %d (%s) is not %s", argNum, arg.Type().String(), sb.String())
}
