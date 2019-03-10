package igor

// ObjectBool is a boolean interpreter type.
type ObjectBool bool

// Type reports the type of the boolean interpreter type.
func (obj ObjectBool) Type() Type {
	return TypeBool
}
