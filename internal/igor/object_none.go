package igor

// ObjectNone is a nil interpreter type.
type ObjectNone struct{}

// Type reports the type of the nil interpreter type.
func (obj ObjectNone) Type() Type {
	return TypeNone
}
