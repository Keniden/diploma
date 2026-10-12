package values

type ReserveStatus int

const (
	ReserveStatusQueued                 ReserveStatus = 0 // Queued
	ReserveStatusWaitingApproval        ReserveStatus = 1 // WaitingApproval
	ReserveStatusActive                 ReserveStatus = 2 // Active
	ReserveStatusExpiredWaitingApproval ReserveStatus = 3 // ExpiredWaitingApproval
	ReserveStatusExpiredActive          ReserveStatus = 4 // ExpiredActive
	ReserveStatusReleased               ReserveStatus = 5 // Released
	ReserveStatusCanceled               ReserveStatus = 6 // Canceled
)

type ReserveGate struct {
	ID             GateID
	Name           string
	PositionNumber int32
}

type ReserveTask struct {
	ID ReserveTaskID
}
