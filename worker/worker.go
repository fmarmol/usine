package worker

import (
	"fmt"
	"log"
	"pool/job"
	"pool/result"
	"pool/rorre"
	"pool/status"
	"time"

	"github.com/google/uuid"
)

// WorkerStatus struct
type WorkerStatus struct {
	ID          uuid.UUID
	IdleTime    time.Duration
	MaxIdleTime time.Duration
	ActiveTime  time.Duration
	Status      status.Status
}

func (ws WorkerStatus) String() string {
	return fmt.Sprintf("WorkerStatus> ID:%v, IdleTime:%v, MaxIdleTime:%v, ActiveTime:%v, Status:%v", ws.ID, ws.IdleTime, ws.MaxIdleTime, ws.ActiveTime, ws.Status)
}

// Worker struct
type Worker struct {
	ID               uuid.UUID
	ChanPoolToWorker chan status.OrderPoolToWorker
	ChanWorkerToPool chan status.OrderWorkerToPoll
	ChanStatus       chan WorkerStatus
	RegisterWorker   chan *Worker
	Jobs             chan *job.Job
	Results          chan result.Result
	Errors           chan rorre.Error
	Ticker           *time.Ticker
	WorkerStatus
}

func NewWorker() *Worker {
	id := uuid.New()
	return &Worker{
		ID:           id,
		WorkerStatus: WorkerStatus{ID: id},
	}

}

// Register: Worker registers him self to pool
func (w *Worker) Register(ch chan *Worker) {
	ch <- w
}

func (w *Worker) Run() {
	log.Printf("Worker ID: %v started\n", w.ID)
	w.Register(w.RegisterWorker)
LOOP:
	for {
		select {
		case order := <-w.ChanPoolToWorker:
			if order == status.PW_STOP && w.Status == status.PENDING {
				w.Status = status.STOPPED
				log.Printf("worker ID: %v is now stopped, status:%v\n", w.ID, w.Status)
				w.ChanWorkerToPool <- status.WP_CONFIRM
				close(w.ChanPoolToWorker)
				close(w.ChanWorkerToPool)
				break LOOP
			}
			if order == status.PW_STATUS {
				log.Printf("worker ID:%v recieved request of status from pool\n", w.ID)
				w.ChanStatus <- w.WorkerStatus
				log.Printf("worker ID:%v send status to pool\n", w.ID)
			}
		case <-w.Ticker.C:
			w.IdleTime += time.Second
			if w.IdleTime > w.MaxIdleTime {
				log.Printf("idletime reached workerID: %v ask to STOP\n", w.ID)
				w.ChanWorkerToPool <- status.WP_STOP
			} else {
				log.Printf("Worker> Id: %v ticked, IdleTime: %v\n", w.ID, w.IdleTime)
			}
		case job := <-w.Jobs:
			w.Status = status.RUNNING
			w.IdleTime = 0
			job.Run(w.Results, w.Errors)
			w.Status = status.PENDING
		}
	}
}
