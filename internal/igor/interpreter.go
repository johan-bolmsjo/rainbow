package igor

import (
	"github.com/johan-bolmsjo/saft"
)

// Function is a function executable by the interpreter.
type Function func(args []Object) Object

// Interpreter is an interpreter instance.
type Interpreter struct {
	functions map[string]Function
}

// NewInterpreter returns a new interpreter.
func NewInterpreter() *Interpreter {
	t := Interpreter{
		functions: map[string]Function{},
	}

	// Register generic logical functions that do not rely on external state.
	t.RegisterFunction("not", func(args []Object) Object {
		if len(args) != 1 {
			Throw(ExceptionInvalidNumberOfArguments(len(args), "1"))
		}
		return ObjectBool(!objectIsTrue(args[0]))
	})

	t.RegisterFunction("and", func(args []Object) Object {
		result := Object(ObjectBool(true))
		for _, arg := range args {
			result = arg
			if !objectIsTrue(arg) {
				break
			}
		}
		return result
	})

	t.RegisterFunction("or", func(args []Object) Object {
		result := Object(ObjectBool(false))
		for _, arg := range args {
			result = arg
			if objectIsTrue(arg) {
				break
			}
		}
		return result
	})

	t.RegisterFunction("equal?", func(args []Object) Object {
		if len(args) != 2 {
			Throw(ExceptionInvalidNumberOfArguments(len(args), "2"))
		}
		return objectIsEqual(args[0], args[1])
	})

	return &t
}

// RegisterFunction registers a function executable by the interpreter.
func (p *Interpreter) RegisterFunction(name string, f Function) {
	p.functions[name] = f
}

// getFunction returns the function registered under name, or nil.
func (p *Interpreter) getFunction(name string) Function {
	return p.functions[name]
}

// CompileCondition compiles a condition.
func (p *Interpreter) CompileCondition(elem saft.Elem) (*Condition, error) {
	call, err := p.compile(elem)
	if err != nil {
		return nil, err
	}
	return &Condition{call: call}, nil
}

// compile compiles a function call expression.
func (p *Interpreter) compile(elem saft.Elem) (*objectCall, error) {
	list, err := elem.ExpectList()
	if err != nil {
		return nil, err
	}

	if len(list.L) == 0 {
		return nil, formatErrorWithPosition(list.Pos(), "missing function name")
	}

	str, ok := list.L[0].IsString()
	if !ok {
		return nil, formatErrorWithPosition(list.L[0].Pos(), "expected function name")
	}

	functionName := str.V

	call := objectCall{
		position: list.Pos(),
		name:     functionName,
		fun:      p.getFunction(functionName),
	}

	if call.fun == nil {
		return nil, formatErrorWithPosition(list.L[0].Pos(), "unknown function %q", functionName)
	}

	for _, arg := range list.L[1:] {
		if str, ok := arg.IsString(); ok {
			call.args = append(call.args, ObjectString(str.V))
		} else if _, ok := arg.IsList(); ok {
			call2, err := p.compile(arg)
			if err != nil {
				return nil, err
			}
			call.args = append(call.args, call2)
		} else {
			return nil, formatErrorWithPosition(arg.Pos(), "expected string or function call")
		}
	}

	return &call, nil
}
