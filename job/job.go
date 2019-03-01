package job

import (
	"context"
	"fmt"
	"time"

	"github.com/fmarmol/usine/result"
	"github.com/fmarmol/usine/rorre"
	"github.com/google/uuid"
)

type function = func(args ...interface{}) (interface{}, error)

// Runable is an interface
type Runable interface {
	Run() (*result.Result, error)
}

// Job struct
type Job struct {
	Name string          // Name of the job
	ID   uuid.UUID       // ID of the job
	F    function        // function that produces an interface and an error
	Args []interface{}   //arguments
	ctx  context.Context // context
}

// String method
func (j Job) String() string {
	return fmt.Sprintf("Job>ID: %v, Name: %v", j.ID, j.Name)
}

// New creates a new job
func New(ctx context.Context, name string, f function, args ...interface{}) *Job {
	return &Job{F: f, ID: uuid.New(), Name: name, Args: args, ctx: ctx}
}

// Run runs the job
func (j *Job) Run() (*result.Result, error) {
	t := time.Now()
	value, err := j.F(j.Args...)
	if err != nil {
		return nil, &rorre.Error{ID: j.ID, Err: err}

	}
	return &result.Result{ID: j.ID, Duration: time.Since(t), Value: value}, nil
}

// Add function example
func Add(ints ...interface{}) (interface{}, error) {
	//time.Sleep(3 * time.Second)
	if len(ints) != 2 {
		return nil, fmt.Errorf("Add function has only 2 arguments")
	}
	return ints[0].(int) + ints[1].(int), nil
}
