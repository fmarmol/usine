package pool

import (
	"sync"
	"time"

	"github.com/fmarmol/usine/job"
	"github.com/fmarmol/usine/result"
	"github.com/fmarmol/usine/rorre"
	"github.com/fmarmol/usine/status"
	"github.com/fmarmol/usine/worker"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
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
	Jobs chan *job.Job
}

func (jm *JobManager) Run(p *Pool) {
	go func() {
		for j := range jm.Jobs {
			p.Jobs <- j
		}
	}()
	for i := 0; i < 30; i++ {
		log.Println("send job")
		jm.Jobs <- job.NewJob(job.Add, "add 3 and 4", p.Results, p.Errors, 3, 4)
		time.Sleep(100 * time.Millisecond)
	}
}

// NewJobManager creates a new Job Manager
func NewJobManager() *JobManager {
	return &JobManager{
		Jobs: make(chan *job.Job),
	}
}

// Pool of workers
type Pool struct {
	Workers        map[uuid.UUID]*worker.Worker
	Min, Max       int
	Jobs           chan *job.Job
	Results        chan *result.Result
	Errors         chan *rorre.Error
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
		Jobs:           make(chan *job.Job),
		Results:        make(chan *result.Result),
		Errors:         make(chan *rorre.Error),
		Close:          make(chan struct{}),
		RegisterWorker: make(chan *worker.Worker),
		ChanCli:        make(chan status.OrderPoolToWorker),
	}
	return p
}

func (p *Pool) NewWorker() *worker.Worker {
	return worker.NewWorker(
		p.RegisterWorker,
		p.Jobs,
		p.Results,
		p.Errors,
		time.Second,
	)
}

func (p *Pool) Init() {
	for i := 0; i < p.Min; i++ {
		go p.NewWorker().Run()
	}
}

func (p *Pool) Run() {
	go func() {
		for {
			select {
			case w := <-p.RegisterWorker:
				log.WithFields(log.Fields{
					"worker": w,
				}).Info("registration")
				go func(worker *worker.Worker) {
					p.Lock()
					p.Workers[w.ID] = w
					p.Unlock()
					defer log.WithFields(log.Fields{"worker": w}).Warn("quit the pool")
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
						case info := <-worker.ChanInfo:
							log.Printf("%+v\n", info)
						}
					}
				}(w)
			case order := <-p.ChanCli:
				if order == status.PW_INFO {
					for _, worker := range p.Workers {
						worker.ChanPoolToWorker <- status.PW_INFO
					}
				}

			case result := <-p.Results:
				log.WithFields(log.Fields{"result": result}).Info("RESULT")
			case err := <-p.Errors:
				log.WithFields(log.Fields{"error": err}).Info("ERROR")
			}
		}
	}()

	go func() {
		for {
			p.Lock()
			count_running := 0
			count_pending := 0
			count_stopped := 0
			for _, worker := range p.Workers {
				switch worker.Status {
				case status.PENDING:
					count_pending += 1
				case status.RUNNING:
					count_running += 1
				case status.STOPPED:
					count_stopped += 1
				}
			}
			p.Unlock()
			time.Sleep(time.Second)
		}
	}()
	<-p.Close
}

func (p *Pool) CreateWorkers(n int) {
	for i := 0; i < n; i++ {
		p.NewWorker()
	}
}
