package igor

// ObjectStringList is a string list interpreter type.
type ObjectStringList []string

// Type reports the type of the string list interpreter type.
func (obj ObjectStringList) Type() Type {
	return TypeStringList
}

func (lhs ObjectStringList) equal(rhs ObjectStringList) ObjectBool {
	if len(rhs) != len(lhs) {
		return false
	}
	for i, v := range rhs {
		if v != lhs[i] {
			return false
		}
	}
	return true
}
