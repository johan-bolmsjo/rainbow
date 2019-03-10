package igor

import (
	"fmt"
	"github.com/johan-bolmsjo/saft"
)

func posErrorf(pos saft.LexPos, format string, a ...interface{}) error {
	return fmt.Errorf(pos.String()+": "+format, a...)
}
