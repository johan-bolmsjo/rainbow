package igor

import (
	"fmt"

	"github.com/johan-bolmsjo/saft"
)

// formatErrorWithPosition returns an error prefixed with the source position.
func formatErrorWithPosition(position saft.LexPos, format string, a ...interface{}) error {
	return fmt.Errorf(position.String()+": "+format, a...)
}
