package igor

// Object is one of the defined interpreter objects.
type Object interface {
	Type() Type
}

func objectIsEqual(lhs, rhs Object) ObjectBool {
	switch lhs := lhs.(type) {
	case ObjectBool:
		if rhs, ok := rhs.(ObjectBool); ok {
			return lhs == rhs
		}
	case ObjectString:
		if rhs, ok := rhs.(ObjectString); ok {
			return lhs == rhs
		}
	case ObjectStringList:
		if rhs, ok := rhs.(ObjectStringList); ok {
			return lhs.equal(rhs)
		}
	}
	return false
}

func objectIsTrue(obj Object) ObjectBool {
	switch obj := obj.(type) {
	case ObjectNone:
		return false
	case ObjectBool:
		return obj
	}
	return true
}

// Type is a type known to the interpreter.
type Type byte

const (
	TypeNone Type = iota
	TypeBool
	TypeCall
	TypeString
	TypeStringList
)

// String converts an interpreter type to a textual representation.
func (t Type) String() string {
	switch t {
	case TypeNone:
		return "None"
	case TypeBool:
		return "Bool"
	case TypeCall:
		return "Call"
	case TypeString:
		return "String"
	case TypeStringList:
		return "StringList"
	}
	return "?"
}
