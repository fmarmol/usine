package job

import (
	"fmt"
	"time"

	"github.com/fmarmol/usine/result"
	"github.com/fmarmol/usine/rorre"
	"github.com/google/uuid"
)

type function = func(args ...interface{}) (interface{}, error)

// Job struct
type Job struct {
	Name    string               // Name of the job
	ID      uuid.UUID            // ID of the job
	F       function             // function that produces an interface and an error
	Results chan<- result.Result // chan of results
	Errors  chan<- rorre.Error   // chan of errors
	Args    []interface{}        //arguments
}

func (j Job) String() string {
	return fmt.Sprintf("Job>ID: %v, Name: %v", j.ID, j.Name)
}

// NewJob creates a new job
func NewJob(f function, name string, r chan<- result.Result, e chan<- rorre.Error, args ...interface{}) *Job {
	return &Job{F: f, ID: uuid.New(), Name: name, Results: r, Errors: e, Args: args}
}

// Run runs the job
func (j *Job) Run() {
	t := time.Now()
	if value, err := j.F(j.Args...); err != nil {
		j.Errors <- rorre.Error{ID: j.ID, Err: err}
	} else {
		j.Results <- result.Result{ID: j.ID, Duration: time.Since(t), Value: value}
	}
}

func Add(ints ...interface{}) (interface{}, error) {
	time.Sleep(3 * time.Second)
	if len(ints) == 2 {
		return ints[0].(int) + ints[1].(int), nil
	}
	return nil, fmt.Errorf("add function should take only 2 arguments")
}
