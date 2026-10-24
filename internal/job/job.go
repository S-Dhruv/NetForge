package job

import (
	"time"
	"errors"
)

type Job struct{
	ID int64
	RunID int64
	Type string
	Payload string
	CurrentState State
	createdAt time.Time
	updatedAt time.Time
	failureReason string
}

// returning a job pointer so that it is modifiable and unique for each call 
func NewJob(runID int64, jobType string, payload string) *Job {
	return &Job{
		ID: time.Now().UnixNano(),
		RunID: runID,
		Type: jobType,
		Payload: payload,
		CurrentState: Waiting,
		failureReason: "None",
		createdAt: time.Now().UTC(),
		updatedAt: time.Now().UTC(),
	}
}

func (j *Job) TransitionTo(next State) error {
	
	current:= j.CurrentState
	possibleTransition := CanTransition(current, next)
	
	if possibleTransition {
		j.CurrentState = next
		j.updatedAt = time.Now().UTC()
		return nil
	}
	j.failureReason = "Invalid Transition"
	return errors.New("ErrInvalidTranistion")

}

