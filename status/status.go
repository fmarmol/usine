package status

type Status string
type OrderPoolToWorker string
type OrderWorkerToPoll string

// Status of worker
const (
	RUNNING Status = "running"
	PENDING Status = "pending"
	STOPPED Status = "stopped"
)

const (
	PW_STOP   OrderPoolToWorker = "stop"
	PW_STATUS                   = "status"
)

const (
	WP_STOP    OrderWorkerToPoll = "stop"
	WP_CONFIRM OrderWorkerToPoll = "confirm"
)
