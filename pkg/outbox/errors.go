package outbox

type DatabaseError struct {
	Err error
}

func (e *DatabaseError) Error() string {
	return "database: " + e.Err.Error()
}

type HandleError struct {
	Message Message
	Err     error
}

func (e *HandleError) Error() string {
	return "handle: " + e.Err.Error()
}

type ScheduleError struct {
	Message Message
	Err     error
}

func (e *ScheduleError) Error() string {
	return "schedule: " + e.Err.Error()
}

type ReadError struct {
	Err error
}

func (e *ReadError) Error() string {
	return "read: " + e.Err.Error()
}

type CompleteError struct {
	Messages []Message
	Err      error
}

func (e *CompleteError) Error() string {
	return "complete: " + e.Err.Error()
}

type DiscardError struct {
	Messages []Message
	Err      error
}

func (e *DiscardError) Error() string {
	return "discard: " + e.Err.Error()
}
