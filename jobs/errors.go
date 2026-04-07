package jobs

import "errors"

// ErrJobNotFound is returned when a job with the given ID does not exist.
var ErrJobNotFound = errors.New("jobs: job not found")
