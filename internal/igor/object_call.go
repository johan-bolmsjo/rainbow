package igor

import (
	"fmt"
	"github.com/johan-bolmsjo/saft"
)

type objectCall struct {
	pos  saft.LexPos
	name string
	fun  Function
	args []Object
}

func (call *objectCall) Type() Type {
	return TypeCall
}

func (call *objectCall) eval() Object {
	var args []Object
	for _, arg := range call.args {
		switch arg := arg.(type) {
		case *objectCall:
			args = append(args, arg.eval())
		default:
			args = append(args, arg)
		}
	}

	defer decorateException(func(err error) error {
		return fmt.Errorf("%s: %s: %s", call.pos.String(), call.name, err)
	})
	return call.fun(args)
}

func (call *objectCall) evalTop() (obj Object, err error) {
	defer catch(&err)
	obj = call.eval()
	return
}
