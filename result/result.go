package result

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Result struct
type Result struct {
	ID       uuid.UUID // keep tracks of job who produce this result
	Duration time.Duration
	Value    interface{}
}

func (r *Result) String() string {
	return fmt.Sprintf("Result>ID: %v, Duration: %v, Value: %v", r.ID, r.Duration, r.Value)
}
