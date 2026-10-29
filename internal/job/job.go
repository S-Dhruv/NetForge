package job

import (
	"errors"
	"time"
)

type Job struct {
	ID            int64
	RunID         int64
	Type          string
	Payload       string
	CurrentState  State
	CreatedAt     time.Time
	UpdatedAt     time.Time
	FailureReason string
}

// returning a job pointer so that it is modifiable and unique for each call
func NewJob(runID int64, jobType string, payload string) *Job {
	return &Job{
		ID:            time.Now().UnixNano(),
		RunID:         runID,
		Type:          jobType,
		Payload:       payload,
		CurrentState:  Waiting,
		FailureReason: "None",
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
	}
}

func (j *Job) TransitionTo(next State) error {

	current := j.CurrentState
	possibleTransition := CanTransition(current, next)

	if possibleTransition {
		j.CurrentState = next
		j.UpdatedAt = time.Now().UTC()
		return nil
	}
	j.FailureReason = "Invalid Transition"
	return errors.New("ErrInvalidTranistion")

}
