package cron

import (
	"time"

	"github.com/robfig/cron/v3"
)

type Schedule interface {
	Previous(time.Time) time.Time
	Next(time.Time) time.Time
}

type schedule struct {
	cron.Schedule
}

func ParseStandard(standardSpec string) (Schedule, error) {
	s, err := cron.ParseStandard(standardSpec)
	if err != nil {
		return nil, err
	}
	return &schedule{Schedule: s}, nil
}

func (s *schedule) Previous(t time.Time) time.Time {
	// Iterate backwards to find the previous time
	prevTime := t
	for {
		prevNextTime := s.Next(prevTime)
		if prevNextTime.Before(t) {
			return prevNextTime
		}
		prevTime = prevTime.Add(-time.Minute)
	}
}
