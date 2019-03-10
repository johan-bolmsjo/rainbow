package igor

import (
	"github.com/johan-bolmsjo/saft"
)

// Function is a function executable by the interpreter.
type Function func(args []Object) Object

// Interp is an interpreter instance.
type Interp struct {
	functions map[string]Function
}

// NewInterp returns a new interpreter.
func NewInterp() *Interp {
	t := Interp{
		functions: map[string]Function{},
	}

	// Register generic logical functions that does not rely on external state.
	t.RegisterFunction("not", func(args []Object) Object {
		if len(args) != 1 {
			Throw(ExceptInvalidNumberOfArgs(len(args), "1"))
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
			Throw(ExceptInvalidNumberOfArgs(len(args), "2"))
		}
		return objectIsEqual(args[0], args[1])
	})

	return &t
}

// RegisterFunction registers a function executable by the interpreter.
func (p *Interp) RegisterFunction(name string, f Function) {
	p.functions[name] = f
}

func (p *Interp) getFunction(name string) Function {
	return p.functions[name]
}

// CompileCond compiles a condition.
func (p *Interp) CompileCond(elem saft.Elem) (*Cond, error) {
	call, err := p.compile(elem)
	if err != nil {
		return nil, err
	}
	return &Cond{call: call}, nil
}

func (p *Interp) compile(elem saft.Elem) (*objectCall, error) {
	list, err := elem.ExpectList()
	if err != nil {
		return nil, err
	}

	if len(list.L) == 0 {
		return nil, posErrorf(list.Pos(), "missing function name")
	}

	str, ok := list.L[0].IsString()
	if !ok {
		return nil, posErrorf(list.L[0].Pos(), "expected function name")
	}

	functionName := str.V

	call := objectCall{
		pos:  list.Pos(),
		name: functionName,
		fun:  p.getFunction(functionName),
	}

	if call.fun == nil {
		return nil, posErrorf(list.L[0].Pos(), "unknown function %q", functionName)
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
			return nil, posErrorf(arg.Pos(), "expected string or function call")
		}
	}

	return &call, nil
}
