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
	args     []Object
}

// Type reports the type of the function call interpreter type.
func (call *objectCall) Type() Type {
	return TypeCall
}

// evaluate evaluates the arguments and invokes the function.
func (call *objectCall) evaluate() Object {
	var args []Object
	for _, arg := range call.args {
		switch arg := arg.(type) {
		case *objectCall:
			args = append(args, arg.evaluate())
		default:
			args = append(args, arg)
		}
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
