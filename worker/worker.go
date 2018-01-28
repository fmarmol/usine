package worker

import (
	"fmt"
	"time"

	"github.com/fmarmol/usine/job"
	"github.com/fmarmol/usine/result"
	"github.com/fmarmol/usine/rorre"
	"github.com/fmarmol/usine/status"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
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

func (w *Worker) String() string {
	return fmt.Sprintf("Worker> ID:%v, IdleTime:%v, MaxIdleTime:%v, ActiveTime:%v, Status:%v", w.ID, w.WorkerStatus.IdleTime, w.WorkerStatus.MaxIdleTime, w.WorkerStatus.ActiveTime, w.WorkerStatus.Status)
}

func NewWorker(r chan *Worker, j chan *job.Job, re chan result.Result, e chan rorre.Error) *Worker {
	id := uuid.New()
	return &Worker{
		ID:               id,
		ChanPoolToWorker: make(chan status.OrderPoolToWorker),
		ChanWorkerToPool: make(chan status.OrderWorkerToPoll),
		ChanStatus:       make(chan WorkerStatus),
		RegisterWorker:   r,
		Jobs:             j,
		Results:          re,
		Errors:           e,
		Ticker:           time.NewTicker(time.Second),
		WorkerStatus: WorkerStatus{
			ID:          id,
			Status:      status.PENDING,
			MaxIdleTime: 5 * time.Second,
		},
	}

}

// Register: Worker registers him self to pool
func (w *Worker) Register(ch chan *Worker) {
	ch <- w
}

func (w *Worker) Run() {
	log.WithFields(log.Fields{"worker": w}).Info("start")
	w.Register(w.RegisterWorker)
LOOP:
	for {
		select {
		case order := <-w.ChanPoolToWorker:
			if order == status.PW_STOP && w.Status == status.PENDING {
				w.Status = status.STOPPED
				log.WithFields(log.Fields{"worker": w}).Warn("stop")
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
				log.WithFields(log.Fields{"worker": w}).Warn("stop idletime > maxidletime")
				w.ChanWorkerToPool <- status.WP_STOP
			} else {
				log.WithFields(log.Fields{"worker": w}).Warn("tick")
			}
		case job := <-w.Jobs:
			w.Status = status.RUNNING
			w.IdleTime = 0
			job.Run(w.Results, w.Errors)
			w.Status = status.PENDING
		}
	}
}
