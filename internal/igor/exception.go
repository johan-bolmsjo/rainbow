package igor

import (
	"fmt"
	"strings"
)

type exception struct {
	err error
}

// Throw exception in interpreter.
func Throw(err error) {
	panic(&exception{err})
}

// Catch eny exception thrown when evaluating expression.
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

// Same as catch but also decorate error using the supplied function.
func catchAndDecorate(err *error, decorate func(err error) error) {
	if x := recover(); x != nil {
		if e, ok := x.(*exception); ok {
			*err = decorate(e.err)
		} else {
			panic(x)
		}
	}
}

// Decorate any thrown excpetion using the specified function.
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

// ExceptInvalidNumberOfArgs is thrown on invalid number of arguments.
func ExceptInvalidNumberOfArgs(args int, expected string) error {
	if expected == "" {
		return fmt.Errorf("invalid number of arguments: %d", args)
	} else {
		return fmt.Errorf("invalid number of arguments: %d, expected: %s", args, expected)
	}
}

// ExceptTypeError is thrown on invalid arguments.
func ExceptInvalidArgument(argNum int, reason string) error {
	return fmt.Errorf("invalid argument: %d, %s", argNum, reason)
}

// ExceptTypeError is thrown on type errors.
func ExceptTypeError(arg Object, argNum int, accepted ...Type) error {
	var sb strings.Builder
	for _, t := range accepted {
		if sb.Len() > 0 {
			sb.WriteByte('|')
		}
		sb.WriteString(t.String())
	}
	return fmt.Errorf("type error: argument %d (%s) is not %s", argNum, arg.Type().String(), sb.String())
}
