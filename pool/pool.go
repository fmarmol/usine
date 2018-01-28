package pool

import (
	"io"
	"log"
	"pool/job"
	"pool/result"
	"pool/rorre"
	"pool/status"
	"pool/worker"
	"sync"
	"time"

	"github.com/google/uuid"
)

type ConfigPool struct {
	Jobs         chan *job.Job
	Results      chan result.Result
	Errors       chan rorre.Error
	Registration chan<- *worker.Worker
	ChanCli      chan status.OrderPoolToWorker
}

// We image a job to do an addition between 2 numbers positives

// JobManager struct
type JobManager struct {
	Reader io.Reader
	Jobs   chan *job.Job
}

// NewJobManager creates a new Job Manager
func NewJobManager(r io.Reader) *JobManager {
	return &JobManager{
		Reader: r,
		Jobs:   make(chan *job.Job),
	}
}

// Pool of workers
type Pool struct {
	Workers        map[uuid.UUID]*worker.Worker
	Min, Max       int
	Jobs           chan *job.Job
	Results        chan result.Result
	Errors         chan rorre.Error
	RegisterWorker chan *worker.Worker
	ChanCli        chan status.OrderPoolToWorker
	Close          chan struct{}
}

// New Pool of workers
func NewPool(min, max int) *Pool {
	p := &Pool{
		Workers:        map[uuid.UUID]*worker.Worker{},
		Min:            min,
		Max:            max,
		Jobs:           make(chan *job.Job),
		Results:        make(chan result.Result),
		Errors:         make(chan rorre.Error),
		Close:          make(chan struct{}),
		RegisterWorker: make(chan *worker.Worker),
		ChanCli:        make(chan status.OrderPoolToWorker),
	}
	return p
}

func (p *Pool) NewWorker() *worker.Worker {
	w := &worker.Worker{
		ChanPoolToWorker: make(chan status.OrderPoolToWorker),
		ChanWorkerToPool: make(chan status.OrderWorkerToPoll),
		ChanStatus:       make(chan worker.WorkerStatus),
		RegisterWorker:   p.RegisterWorker,
		Jobs:             p.Jobs,
		Results:          p.Results,
		Errors:           p.Errors,
		Ticker:           time.NewTicker(time.Second),
	}
	w.ID = uuid.New()
	w.Status = status.PENDING
	w.MaxIdleTime = 5 * time.Second
	//p.Workers = append(p.Workers, w)
	return w
}

func (p *Pool) Init() {
	for i := 0; i < p.Min; i++ {
		go p.NewWorker().Run()
	}
}

func (p *Pool) Run() {
	mux := sync.RWMutex{}
	go func() {
		for {
			select {
			case w := <-p.RegisterWorker:
				log.Printf("pool received a request for registration of worker ID:%v \n", w.ID)
				go func(worker *worker.Worker) {
					mux.Lock()
					p.Workers[w.ID] = w
					mux.Unlock()
					log.Printf("Worker ID: %v is now registered\n", worker.ID)
					defer log.Printf("Worker Id: %v is not yet in the pool\n", worker.ID)
				LOOP:
					for {
						select {
						case order := <-worker.ChanWorkerToPool:
							switch order {
							case status.WP_STOP:
								log.Printf("pool say to worker %v stop\n", worker.ID)
								worker.ChanPoolToWorker <- status.PW_STOP
								<-worker.ChanWorkerToPool
								break LOOP
							}
						case status := <-worker.ChanStatus:
							log.Printf("%+v\n", status)
						}
					}
				}(w)
			case order := <-p.ChanCli:
				if order == status.PW_STATUS {
					log.Println("pool recieved status request")
					for _, worker := range p.Workers {
						log.Println("send status to", worker.ID)
						worker.ChanPoolToWorker <- status.PW_STATUS
					}
				}
			}
		}
	}()
	<-p.Close
}

func (p *Pool) CreateWorkers(n int) {
	for i := 0; i < n; i++ {
		p.NewWorker()
	}
}
