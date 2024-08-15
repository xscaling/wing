package timerange

import "time"

type Scheduler interface {
	Contains(when time.Time) bool
	GetStart() string
	GetEnd() string
	GetTimezone() *time.Location
}

type baseScheduler struct {
	timezone *time.Location
	rawStart string
	rawEnd   string
}

func (s baseScheduler) GetStart() string {
	return s.rawStart
}

func (s baseScheduler) GetEnd() string {
	return s.rawEnd
}

func (s baseScheduler) GetTimezone() *time.Location {
	return s.timezone
}
