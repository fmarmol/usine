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
	go func() {
		fmt.Println("fetch register")
		<-worker.RegisterWorker
		worker.ChanPoolToWorker <- status.PW_CONFIRM
		fmt.Println("confirm sent")
	}()

	go func() {
		for r := range worker.Results {
			fmt.Println(r)
		}
	}()

	go func() {
		for r := range worker.Errors {
			fmt.Println(r)
		}
	}()

	go func() {
		select {
		case order := <-worker.ChanWorkerToPool:
			log.Printf("Worker says: %T, %+v\n", order, order)
			switch order {
			case status.WP_STOP:
				log.Println("Pool repoonds to worker with:", status.PW_STOP)
				worker.ChanPoolToWorker <- status.PW_STOP
			}

		}
	}()
	worker.Run()
}
