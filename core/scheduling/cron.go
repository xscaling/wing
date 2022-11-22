package scheduling

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
)

var (
	ErrCronScheduleSupportsExactMinuteHourValueOnly = errors.New("cron schedule supports exact minute and hour value only")
	ErrNotAStandardCronSpec                         = errors.New("not a standard cron spec(https://en.wikipedia.org/wiki/Cron)")
)

const (
	cronAnyRangeListCharacters = "*/-,"
	cronFieldSeparator         = " "
)

func validateCronSpec(spec string) error {
	if spec[0] != '@' {
		fields := strings.Split(spec, cronFieldSeparator)
		if len(fields) != 5 {
			return fmt.Errorf("%w: `%s`", ErrNotAStandardCronSpec, spec)
		}
		if strings.ContainsAny(fields[0], cronAnyRangeListCharacters) ||
			strings.ContainsAny(fields[1], cronAnyRangeListCharacters) {
			return fmt.Errorf("%w: `%s`", ErrCronScheduleSupportsExactMinuteHourValueOnly, spec)
		}
	}
	_, err := cron.ParseStandard(spec)
	return err
}

func parseCronScheduleSpec(spec string) (cron.Schedule, error) {
	if err := validateCronSpec(spec); err != nil {
		return nil, err
	}
	sched, err := cron.ParseStandard(spec)
	return sched, err
}

type CronScheduler struct {
	baseScheduler
	startSched cron.Schedule
	endSched   cron.Schedule
}

func NewCronScheduler(timezone *time.Location, start, end string) (*CronScheduler, error) {
	s := &CronScheduler{
		baseScheduler: baseScheduler{
			timezone: timezone,
			rawStart: start,
			rawEnd:   end,
		},
	}
	var err error
	s.startSched, err = parseCronScheduleSpec(start)
	if err != nil {
		return nil, err
	}
	s.endSched, err = parseCronScheduleSpec(end)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (s *CronScheduler) GetUpcomingTriggerDuration(when time.Time) (start, end time.Time) {
	whenInTimezone := when.In(s.timezone)
	return s.startSched.Next(whenInTimezone), s.endSched.Next(whenInTimezone)
}

func (s *CronScheduler) Contains(when time.Time) bool {
	whenInTimezone := when.In(s.timezone)

	start, end := s.GetUpcomingTriggerDuration(whenInTimezone)
	var (
		nextStartTimestamp = start.Unix()
		nextEndTimestamp   = end.Unix()
		currentTimestamp   = whenInTimezone.Unix()
	)
	// current timestamp always before next start timestamp
	// so if current timestamp is before next end timestamp(strong requirement) and next start timestamp is after next end timestamp
	// then current timestamp is in the range of duration.
	// In short, current timestamp before end but already started.
	return nextStartTimestamp > nextEndTimestamp && currentTimestamp <= nextEndTimestamp
}
