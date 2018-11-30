package worker

import (
	"testing"
	"time"

	"github.com/fmarmol/usine/rorre"
	"github.com/fmarmol/usine/status"

	"github.com/fmarmol/usine/job"
	"github.com/fmarmol/usine/result"
)

func TestWorker(t *testing.T) {

	r := make(chan *Worker)
	j := make(chan *job.Job)
	re := make(chan *result.Result)
	e := make(chan *rorre.Error)

	worker := NewWorker(r, j, re, e, time.Second)
	go func() {
		worker.ChanPoolToWorker <- status.PW_CONFIRM
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
