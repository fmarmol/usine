package worker

import (
	"fmt"
	"sync"
	"time"

	"github.com/fmarmol/usine/job"
	"github.com/fmarmol/usine/logger"
	"github.com/fmarmol/usine/result"
	"github.com/fmarmol/usine/rorre"
	"github.com/fmarmol/usine/status"
	"github.com/google/uuid"
)

var log = logger.NewLogger()

// Info struct
type Info struct {
	ID          uuid.UUID
	IdleTime    time.Duration
	MaxIdleTime time.Duration
	ActiveTime  time.Duration
	Status      status.Status
}

func (ws Info) String() string {
	return fmt.Sprintf("WorkerStatus> ID:%v, IdleTime:%v, MaxIdleTime:%v, ActiveTime:%v, Status:%v", ws.ID, ws.IdleTime, ws.MaxIdleTime, ws.ActiveTime, ws.Status)
}

// Worker struct
type Worker struct {
	ID               uuid.UUID
	ChanPoolToWorker chan status.OrderPoolToWorker
	ChanWorkerToPool chan status.OrderWorkerToPool
	ChanInfo         chan Info
	RegisterWorker   chan *Worker
	Jobs             chan *job.Job
	Results          chan *result.Result
	Errors           chan *rorre.Error
	tickTime         time.Duration
	Ticker           *time.Ticker
	CloseOrder       chan struct{}
	CloseJob         chan struct{}
	CloseTick        chan struct{}
	Info
}

func (w *Worker) String() string {
	return fmt.Sprintf("Worker> ID:%v, IdleTime:%v, MaxIdleTime:%v, ActiveTime:%v, Status:%v", w.ID, w.Info.IdleTime, w.Info.MaxIdleTime, w.Info.ActiveTime, w.Info.Status)
}

// NewWorker creates a new worker
func NewWorker(r chan *Worker, j chan *job.Job, re chan *result.Result, e chan *rorre.Error, tickTime time.Duration) *Worker {
	id := uuid.New()
	return &Worker{
		ID:               id,
		ChanPoolToWorker: make(chan status.OrderPoolToWorker),
		ChanWorkerToPool: make(chan status.OrderWorkerToPool),
		ChanInfo:         make(chan Info),
		CloseJob:         make(chan struct{}),
		CloseOrder:       make(chan struct{}),
		CloseTick:        make(chan struct{}),
		RegisterWorker:   r,
		Jobs:             j,
		Results:          re,
		Errors:           e,
		tickTime:         tickTime,
		Ticker:           time.NewTicker(tickTime),
		Info: Info{
			ID:          id,
			Status:      status.PENDING,
			MaxIdleTime: 5 * time.Second,
		},
	}

}

// Register : Worker registers him self to pool
func (w *Worker) Register() {
	w.RegisterWorker <- w
}

// stopJobs private method
func (w *Worker) stopJobs() {
	close(w.Jobs)
	w.CloseJob <- struct{}{}
}

// stopTicker private method
func (w *Worker) stopTicker() {
	w.Ticker.Stop()
	w.CloseTick <- struct{}{}
}

// stopOrders private method
func (w *Worker) stopOrders() {
	close(w.ChanPoolToWorker)
	w.CloseOrder <- struct{}{}
}

// Stop send close signal to all channels and close it
func (w *Worker) Stop() {
	log.Println("Worker send stop to job")
	w.stopJobs()
	log.Println("Worker send stop to order")
	w.stopOrders()
	log.Println("Worker send stop to tick")
	w.stopTicker()
}

// startOrderLoop private method
func (w *Worker) startOrderLoop() error {
	defer func() {
		log.Debugf("worker ID:%v job's loop end", w.ID)
	}()
LOOP:
	for {
		select {
		case order := <-w.ChanPoolToWorker:
			switch order {
			case status.PW_STOP:
				w.Status = status.STOPPED
				log.Infof("Worker ID: %v stopping...\n", w.ID)
				go w.Stop()
			case status.PW_INFO:
				log.Debugf("worker ID:%v recieved request of status from pool\n", w.ID)
				w.ChanInfo <- w.Info
				log.Debugf("worker ID:%v send status to pool\n", w.ID)
			}
		case <-w.CloseOrder:
			break LOOP

		}
	}

	log.Debug("Waiting for order loop to close...")
	return nil
}

// startJobLoop private method
func (w *Worker) startJobLoop() error {
	defer func() {
		log.Debug("Job loop closed")
	}()
LOOP:
	for {
		select {
		case job := <-w.Jobs:
			log.Debugf("Worker ID: %v recieved job: %v", w.ID, job)
			w.Status, w.IdleTime = status.RUNNING, 0
			result, err := job.Run()
			if err != nil {
				w.Errors <- err
			} else {
				w.Results <- result
			}
			w.Status = status.PENDING
		case <-w.CloseJob:
			log.Debug("Waiting for job loop to close...")
			break LOOP
		}
	}
	return nil
}

// startTickerLoop private method
func (w *Worker) startTickerLoop() error {
	defer func() {
		log.Println("Tick loop ended")
	}()
	askToStop := false
LOOP:
	for {
		select {
		case <-w.Ticker.C:
			if w.Status == status.PENDING {
				w.IdleTime += w.tickTime
			}
			if w.IdleTime > w.MaxIdleTime && !askToStop {
				log.Warnf("Worker ID: %v will stop because idletime > maxidletime\n", w.ID)
				w.ChanWorkerToPool <- status.WP_STOP // worker asks the pool to stop
				askToStop = true
			} else {
				log.Debugf("Worker ID: %v tick\n", w.ID)
			}
		case <-w.CloseTick:
			log.Debug("Waiting for tick loop to close...")
			break LOOP
		}
	}
	return nil
}

// Run method to run a worker
func (w *Worker) Run() error {
	defer func() {
		log.Debugf("Worker ID: %v run's ended\n", w.ID)
	}()
	log.Debugf("Worker ID: %v started\n", w.ID)
	go func() {
		w.Register()
	}()
	// TODO Wait for confirmation of registration
	if order := <-w.ChanPoolToWorker; order != status.PW_CONFIRM {
		return fmt.Errorf("Pool should have confirmed registration of worker ID: %v", w.ID)
	}
	wg := sync.WaitGroup{}
	wg.Add(3)
	go func() {
		defer wg.Done()
		w.startOrderLoop()
	}()
	go func() {
		defer wg.Done()
		w.startJobLoop()
	}()
	go func() {
		defer wg.Done()
		w.startTickerLoop()
	}()
	wg.Wait()
	return nil
}
