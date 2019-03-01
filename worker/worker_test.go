package worker

import (
	"fmt"
	"testing"
	"time"

	"github.com/fmarmol/usine/status"

	"github.com/fmarmol/usine/job"
	"github.com/fmarmol/usine/result"
)

func TestWorker(t *testing.T) {

	r := make(chan *Worker)
	j := make(chan job.Runable)
	re := make(chan *result.Result)
	e := make(chan error)

	go func() {
		for i := 0; i < 10; i++ {
			j <- job.New(fmt.Sprintf("Add %v and %v", i, i+1), job.Add, i, i+1)
			fmt.Println("job sent")
		}
	}()

	worker := New(r, j, re, e, time.Second)

	// fake pool that auto confirm any request
	go func() {
		worker.ChanPoolToWorker <- status.PW_CONFIRM
	}()

	// go routine to see results
	go func() {
		for r := range worker.Results {
			fmt.Println(r)
		}
	}()

	// go routine to see errors
	go func() {
		for r := range worker.Errors {
			fmt.Println(r)
		}
	}()
	log.Println("worker run:")
	worker.Run()

}
