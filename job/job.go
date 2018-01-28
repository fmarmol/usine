package job

import (
	"fmt"
	"pool/result"
	"pool/rorre"
	"time"

	"github.com/google/uuid"
)

type function = func() (interface{}, error)

// Job struct
type Job struct {
	Name string    // Name of the job
	ID   uuid.UUID // ID of the job
	F    function  // function that produces an interface and an error
}

func (j Job) String() string {
	return fmt.Sprintf("Job>ID: %v, Name: %v", j.ID, j.Name)
}

// NewJob creates a new job
func NewJob(f function, name string) *Job {
	return &Job{F: f, ID: uuid.New(), Name: name}
}

// Run runs the job
func (j *Job) Run(results chan<- result.Result, errors chan<- rorre.Error) {
	t := time.Now()
	if value, err := j.F(); err != nil {
		errors <- rorre.Error{ID: j.ID, Err: err}
	} else {
		results <- result.Result{ID: j.ID, Duration: time.Since(t), Value: value}
	}
}
