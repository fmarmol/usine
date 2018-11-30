package status

// Status : worker's status
type Status string

// OrderPoolToWorker : order from pool to worker
type OrderPoolToWorker string

// OrderWorkerToPool : order from worker to pool
type OrderWorkerToPool string

// Status of worker
const (
	RUNNING Status = "running" // default value
	PENDING Status = "pending"
	STOPPED Status = "stopped"
)

const (
	PW_STOP    OrderPoolToWorker = "stop"
	PW_INFO    OrderPoolToWorker = "info"
	PW_CONFIRM OrderPoolToWorker = "confirm" // pool confirm an order from worker
)

const (
	WP_STOP    OrderWorkerToPool = "stop"
	WP_CONFIRM OrderWorkerToPool = "confirm" // worker confirm an order from pool
)
