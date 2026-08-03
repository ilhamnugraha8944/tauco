package domain

import (
	"encoding/json"
	"errors"
	"time"
)

type Job struct {
	ID          string
	Kind        string
	Payload     json.RawMessage
	Attempts    int
	MaxAttempts int
	LockedAt    time.Time
	LeaseUntil  time.Time
}

func (job Job) Validate() error {
	if job.ID == "" || job.Kind == "" || len(job.Payload) == 0 {
		return errors.New("job identity, kind, and payload are required")
	}
	if job.Attempts < 1 || job.MaxAttempts < 1 || job.Attempts > job.MaxAttempts {
		return errors.New("invalid job attempts")
	}
	if job.LockedAt.IsZero() || !job.LeaseUntil.After(job.LockedAt) {
		return errors.New("invalid job lease")
	}
	return nil
}
