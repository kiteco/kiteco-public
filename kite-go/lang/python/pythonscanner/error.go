package pythonscanner

import (
	"fmt"
	"go/token"
)

// PosError is an error associated with a position in the buffer.
type PosError struct {
	Pos token.Pos
	Msg string
}

// String converts the error to a string
func (e PosError) Error() string {
	return fmt.Sprintf("%d: %s", e.Pos, e.Msg)
}
