package job

import "errors"

type ErrorCategory string

const (
	NoError        ErrorCategory = "NoError"
	ErrorPermanent ErrorCategory = "NoRetry"
	ErrorTemporary ErrorCategory = "Retry"
)

var ErrNetworkTimeout = errors.New("network timeout")
var JobCreationError = errors.New("Issue with job creation")

func ClassifyError(err error) ErrorCategory {
	if err == nil {
		return NoError
	}

	if errors.Is(err, JobCreationError) || errors.Is(err, ErrNetworkTimeout) {
		return ErrorTemporary
	} else {
		return ErrorPermanent
	}
}
