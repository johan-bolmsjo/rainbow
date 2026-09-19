package igor

// Condition is a condition that evaluates to a boolean value.
type Condition struct {
	call *objectCall
}

// Evaluate evaluates the condition and reports its result.
// An error is reported on failures to execute the condition.
func (cond *Condition) Evaluate() (bool, error) {
	if cond == nil {
		return true, nil
	}
	res, err := cond.call.evaluateTop()
	if err != nil {
		return false, err
	}
	return bool(objectIsTrue(res)), nil
}
