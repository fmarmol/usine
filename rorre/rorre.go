package rorre

import (
	"fmt"

	"github.com/google/uuid"
)

// Error struct
type Error struct {
	ID  uuid.UUID // keep track of job'ID thar produced this error
	Err error
}

func (e Error) String() string {
	return fmt.Sprintf("Error>ID: %v, Error: %v", e.ID, e.Err)
}
