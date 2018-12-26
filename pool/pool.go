package pool

import (
	"fmt"
	"sync"
	"time"

	"github.com/fmarmol/usine/job"
	"github.com/fmarmol/usine/logger"
	"github.com/fmarmol/usine/result"
	"github.com/fmarmol/usine/status"
	"github.com/fmarmol/usine/worker"
	"github.com/google/uuid"
)

var log = logger.NewLogger()

// ConfigPool is the struct of configuration
type ConfigPool struct {
	Jobs         chan job.Runable
	Results      chan result.Result
	Errors       chan error
	Registration chan<- *worker.Worker
	ChanCli      chan status.OrderPoolToWorker
}

// We image a job to do an addition between 2 numbers positives

// JobManager struct
type JobManager struct {
	Jobs chan job.Runable
}

// Run start jobs
func (jm *JobManager) Run(p *Pool) {
	go func() {
		for j := range jm.Jobs {
			p.Jobs <- j
		}
	}()
	for i := 0; i < 10; i++ {
		jm.Jobs <- job.New(fmt.Sprintf("Add %v and %v", i, i+1), job.Add, i, i+1)
		//time.Sleep(100 * time.Millisecond)
	}
}

// NewJobManager creates a new Job Manager
func NewJobManager() *JobManager {
	return &JobManager{
		Jobs: make(chan job.Runable),
	}
}

// Pool of workers
type Pool struct {
	Workers        map[uuid.UUID]*worker.Worker
	Status         map[status.Status][]*worker.Worker
	Min, Max       int
	Jobs           chan job.Runable
	Results        chan *result.Result
	Errors         chan error
	RegisterWorker chan *worker.Worker
	ChanCli        chan status.OrderPoolToWorker
	Close          chan struct{}
	sync.RWMutex
}

// NewPool of workers
func NewPool(min, max int) *Pool {
	p := &Pool{
		Workers:        map[uuid.UUID]*worker.Worker{},
		Min:            min,
		Max:            max,
		Jobs:           make(chan job.Runable),
		Results:        make(chan *result.Result),
		Errors:         make(chan error),
		Close:          make(chan struct{}),
		RegisterWorker: make(chan *worker.Worker),
		ChanCli:        make(chan status.OrderPoolToWorker),
	}
	return p
}

// NewWorker retuns a a new worker
func (p *Pool) NewWorker() *worker.Worker {
	return worker.New(
		p.RegisterWorker,
		p.Jobs,
		p.Results,
		p.Errors,
		time.Second,
	)
}

// Init the pool
func (p *Pool) Init() {
	for i := 0; i < p.Min; i++ {
		go p.NewWorker().Run()
	}
}

// Manage  The pool interacts with a worker
func (p *Pool) Manage(worker *worker.Worker) error {
LOOP:
	for {
		select {
		case order := <-worker.ChanWorkerToPool:
			switch order {
			case status.WP_STOP:
				log.Infof("Pool say to worker %v stop\n", worker.ID)
				worker.ChanPoolToWorker <- status.PW_STOP
				p.Lock()
				delete(p.Workers, worker.ID) // we remove worker from pool
				p.Unlock()
				break LOOP
			}
		case info := <-worker.ChanInfo:
			log.Infof("Worker %v Info: %+v\n", worker, info)
		}
	}
	return nil
}

// Register a new worker
func (p *Pool) Register(worker *worker.Worker) error {
	p.Lock()
	p.Workers[worker.ID] = worker
	p.Unlock()
	worker.ChanPoolToWorker <- status.PW_CONFIRM
	return nil
}

// Run the pool
func (p *Pool) Run() {
	go func() {
		for {
			select {
			case w := <-p.RegisterWorker:
				log.Infof("Worker %v registration\n", w)
				p.Register(w)
				go p.Manage(w)
			case order := <-p.ChanCli:
				if order == status.PW_INFO {
					for _, worker := range p.Workers {
						worker.ChanPoolToWorker <- status.PW_INFO
					}
				}
			case result := <-p.Results:
				log.Infof("Result: %v\n", result)
			case err := <-p.Errors:
				log.Errorf("Error %v \n", err)
			}
		}
	}()

	go func() {
		for {
			countPending := 0
			countRunning := 0
			countStopped := 0

			p.Lock()
			for _, worker := range p.Workers {
				switch worker.GetStatus() {
				case status.PENDING:
					countPending++
				case status.RUNNING:
					countRunning++
				case status.STOPPED:
					countStopped++
				}
			}
			p.Unlock()
			fmt.Println("Pending:", countPending)
			fmt.Println("Running:", countRunning)
			fmt.Println("Stopped:", countStopped)
			fmt.Println("Total:", countPending+countRunning+countStopped)
			time.Sleep(time.Second)
		}
	}()
	<-p.Close
}
