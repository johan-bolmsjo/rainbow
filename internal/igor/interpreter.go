package igor

import (
	"github.com/johan-bolmsjo/saft"
)

// Function is a function executable by the interpreter. An eager function
// receives fully evaluated arguments. A lazy function receives its unevaluated
// arguments and controls when they are evaluated, which allows short-circuiting
// functions such as and and or.
type Function func(args []Object) Object

// registeredFunction is an interpreter function together with how its arguments
// are evaluated.
type registeredFunction struct {
	function Function
	lazy     bool
}

// Interpreter is an interpreter instance.
type Interpreter struct {
	functions map[string]registeredFunction
}

// NewInterpreter returns a new interpreter.
func NewInterpreter() *Interpreter {
	t := Interpreter{
		functions: map[string]registeredFunction{},
	}

	// Register generic logical functions that do not rely on external state.
	t.RegisterEagerFunction("not", func(args []Object) Object {
		if len(args) != 1 {
			Throw(ExceptionInvalidNumberOfArguments(len(args), "1"))
		}
		return ObjectBool(!objectIsTrue(args[0]))
	})

	t.RegisterLazyFunction("and", func(args []Object) Object {
		result := Object(ObjectBool(true))
		for _, arg := range args {
			result = evaluateCallArgument(arg)
			if !objectIsTrue(result) {
				break
			}
		}
		return result
	})

	t.RegisterLazyFunction("or", func(args []Object) Object {
		result := Object(ObjectBool(false))
		for _, arg := range args {
			result = evaluateCallArgument(arg)
			if objectIsTrue(result) {
				break
			}
		}
		return result
	})

	t.RegisterEagerFunction("equal?", func(args []Object) Object {
		if len(args) != 2 {
			Throw(ExceptionInvalidNumberOfArguments(len(args), "2"))
		}
		return objectIsEqual(args[0], args[1])
	})

	return &t
}

// RegisterEagerFunction registers an eager function executable by the
// interpreter. It receives fully evaluated arguments.
func (p *Interpreter) RegisterEagerFunction(name string, f Function) {
	p.functions[name] = registeredFunction{function: f}
}

// RegisterLazyFunction registers a lazy function executable by the interpreter.
// It receives its unevaluated arguments.
func (p *Interpreter) RegisterLazyFunction(name string, f Function) {
	p.functions[name] = registeredFunction{function: f, lazy: true}
}

// getFunction returns the function registered under name and reports whether it
// was found.
func (p *Interpreter) getFunction(name string) (registeredFunction, bool) {
	f, ok := p.functions[name]
	return f, ok
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

	registered, ok := p.getFunction(functionName)
	if !ok {
		return nil, formatErrorWithPosition(list.L[0].Pos(), "unknown function %q", functionName)
	}

	call := objectCall{
		position: list.Pos(),
		name:     functionName,
		fun:      registered.function,
		lazy:     registered.lazy,
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
