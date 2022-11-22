package scheduling

import (
	"errors"
	"fmt"
	"time"
)

var (
	ErrStartDateMustBeBeforeEndDate = errors.New("start date must be before end date")
)

const (
	schedulePeriodDateFormat = "2006-01-02 15:04"
)

type DateScheduler struct {
	baseScheduler
	startTime time.Time
	endTime   time.Time
}

func NewDateScheduler(timezone *time.Location, start, end string) (*DateScheduler, error) {
	s := &DateScheduler{
		baseScheduler: baseScheduler{
			timezone: timezone,
			rawStart: start,
			rawEnd:   end,
		},
	}
	var err error
	s.startTime, err = time.ParseInLocation(schedulePeriodDateFormat, start, timezone)
	if err != nil {
		return nil, fmt.Errorf("%w: start(%s) %s", ErrInvalidSchedulePeriodFormat, start, err)
	}
	s.endTime, err = time.ParseInLocation(schedulePeriodDateFormat, end, timezone)
	if err != nil {
		return nil, fmt.Errorf("%w: end(%s) %s", ErrInvalidSchedulePeriodFormat, end, err)
	}
	if !s.startTime.Before(s.endTime) {
		return nil, fmt.Errorf("%w: start(%s) must be before end(%s)", ErrStartDateMustBeBeforeEndDate, start, end)
	}
	return s, nil
}

func (s *DateScheduler) Contains(when time.Time) bool {
	// start <= when <= end
	whenInTimezone := when.In(s.timezone)
	return !s.startTime.After(whenInTimezone) && !s.endTime.Before(whenInTimezone)
}

func (s *DateScheduler) GetUpcomingTriggerDuration(when time.Time) (time.Time, time.Time) {
	whenInTimezone := when.In(s.timezone)

	// If it's after start time, just return zero time means never trigger this scheduler again
	if whenInTimezone.After(s.startTime) {
		return time.Time{}, time.Time{}
	}
	return s.startTime, s.endTime
}

func (s *DateScheduler) GetEndTime() time.Time {
	return s.endTime
}
