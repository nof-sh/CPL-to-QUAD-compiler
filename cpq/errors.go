package cpq

import "fmt"

//import (
//	"fmt"

//	"github.com/alongubkin/cpl-compiler/pkg/lexer"
//)

// Error represents an error that occurred during code generation.
type Error struct {
	Message string
	Pos     Position
}

// Error returns the string representation of the error.
func (e *Error) Error() string {
	return fmt.Sprintf("%s at line %d, char %d", e.Message, e.Pos.Line+1, e.Pos.Column+1)
}
