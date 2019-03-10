package igor

// Cond is a condition that evaluates to a boolean value.
type Cond struct {
	call *objectCall
}

// Eval evaluates the condition and reports its result.
// An error is reported on failures to execute the condition.
func (cond *Cond) Eval() (bool, error) {
	if cond == nil {
		return true, nil
	}
	res, err := cond.call.evalTop()
	if err != nil {
		return false, err
	}
	return bool(objectIsTrue(res)), nil
}
