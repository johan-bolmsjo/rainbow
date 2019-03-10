package igor

// ObjectString is a string interpreter type.
type ObjectString string

// Type reports the type of the string interpreter type.
func (obj ObjectString) Type() Type {
	return TypeString
}
