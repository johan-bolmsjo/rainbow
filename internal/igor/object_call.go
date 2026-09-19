package igor

import (
	"fmt"

	"github.com/johan-bolmsjo/saft"
)

// objectCall is a compiled function call expression.
type objectCall struct {
	position saft.LexPos
	name     string
	fun      Function
	lazy     bool
	args     []Object
}

// Type reports the type of the function call interpreter type.
func (call *objectCall) Type() Type {
	return TypeCall
}

// evaluateCallArgument evaluates a function call argument. Nested function
// calls are evaluated recursively while plain objects are returned as is.
func evaluateCallArgument(arg Object) Object {
	if call, ok := arg.(*objectCall); ok {
		return call.evaluate()
	}
	return arg
}

// evaluate evaluates the arguments and invokes the function.
func (call *objectCall) evaluate() Object {
	if call.lazy {
		return call.fun(call.args)
	}

	var args []Object
	for _, arg := range call.args {
		args = append(args, evaluateCallArgument(arg))
	}

	defer decorateException(func(err error) error {
		return fmt.Errorf("%s: %s: %s", call.position.String(), call.name, err)
	})
	return call.fun(args)
}

// evaluateTop evaluates the call and reports a thrown exception as an error.
func (call *objectCall) evaluateTop() (obj Object, err error) {
	defer catch(&err)
	obj = call.evaluate()
	return
}
