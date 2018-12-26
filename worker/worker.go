package worker

import (
	"fmt"
	"sync"
	"time"

	"github.com/fmarmol/usine/job"
	"github.com/fmarmol/usine/logger"
	"github.com/fmarmol/usine/result"
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
	sync.Mutex
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
	Jobs             chan job.Runable
	Results          chan *result.Result
	Errors           chan error
	tickTime         time.Duration
	Ticker           *time.Ticker
	CloseOrder       chan struct{}
	CloseJob         chan struct{}
	CloseTick        chan struct{}
	Info
}

// GetStatus returns status thread safe
func (w *Worker) GetStatus() status.Status {
	w.Lock()
	s := w.Status
	w.Unlock()
	return s
}

// SetStatus set status thread safe
func (w *Worker) SetStatus(s status.Status) {
	w.Lock()
	w.Status = s
	w.Unlock()
}

// GetIdleTime returns idle time thread safe
func (w *Worker) GetIdleTime() time.Duration {
	w.Lock()
	d := w.IdleTime
	w.Unlock()
	return d
}

// SetIdleTime sets idle time thread safe
func (w *Worker) SetIdleTime(d time.Duration) {
	w.Lock()
	w.IdleTime = d
	w.Unlock()
}

func (w *Worker) String() string {
	return fmt.Sprintf("Worker> ID:%v, IdleTime:%v, MaxIdleTime:%v, ActiveTime:%v, Status:%v", w.ID, w.Info.IdleTime, w.Info.MaxIdleTime, w.Info.ActiveTime, w.Info.Status)
}

// New creates a new worker
func New(r chan *Worker, j chan job.Runable, re chan *result.Result, e chan error, tickTime time.Duration) *Worker {
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
func (w *Worker) Register() error {
	w.RegisterWorker <- w
	if order := <-w.ChanPoolToWorker; order != status.PW_CONFIRM {
		return fmt.Errorf("Pool should have confirmed registration of worker ID: %v", w.ID)
	}
	return nil
}

// stopJobs private method
func (w *Worker) stopJobs() {
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
	log.Debugln("Worker send stop to job")
	w.stopJobs()
	log.Debugln("Worker send stop to order")
	w.stopOrders()
	log.Debugln("Worker send stop to tick")
	w.stopTicker()
}

// startOrderLoop private method
func (w *Worker) startOrderLoop() error {
	defer func() {
		log.Debugf("worker ID:%v job's loop end\n", w.ID)
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
				log.Debugf("Worker ID:%v recieved request of status from pool\n", w.ID)
				w.ChanInfo <- w.Info
				log.Debugf("Worker ID:%v send status to pool\n", w.ID)
			}
		case <-w.CloseOrder:
			break LOOP

		}
	}

	log.Debugln("Waiting for order loop to close...")
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
		case job, ok := <-w.Jobs:
			if ok && w.GetStatus() == status.PENDING {
				log.Debugf("Worker ID: %v recieved job: %v, %v", w.ID, job, ok)
				w.SetStatus(status.RUNNING)
				w.SetIdleTime(0)
				result, err := job.Run()
				if err != nil {
					w.Errors <- err
				} else {
					w.Results <- result
				}
				w.SetStatus(status.PENDING)
			}
		case <-w.CloseJob:
			log.Debugln("Waiting for job loop to close...")
			break LOOP
		}
	}
	return nil
}

// startTickerLoop private method
func (w *Worker) startTickerLoop() error {
	defer func() {
		log.Debugln("Tick loop ended")
	}()
	askToStop := false
LOOP:
	for {
		select {
		case <-w.Ticker.C:
			if w.GetStatus() == status.PENDING {
				w.SetIdleTime(w.GetIdleTime() + w.tickTime)
			}
			if w.IdleTime > w.MaxIdleTime && !askToStop {
				log.Warnf("Worker ID: %v will stop because idletime > maxidletime\n", w.ID)
				w.ChanWorkerToPool <- status.WP_STOP // worker asks the pool to stop
				askToStop = true
			} else {
				fmt.Println("tick")
			}
		case <-w.CloseTick:
			log.Debugln("Waiting for tick loop to close...")
			break LOOP
		}
	}
	return nil
}

// Run method to run a worker
func (w *Worker) Run() (err error) {
	defer func() {
		log.Debugf("Worker ID: %v run's ended\n", w.ID)
	}()
	log.Debugf("Worker ID: %v started\n", w.ID)
	if err := w.Register(); err != nil {
		return err
	}
	// TODO Wait for confirmation of registration
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
