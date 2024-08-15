package timerange

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/xscaling/wing/utils/cron"
)

var (
	ErrCronScheduleSupportsExactMinuteHourValueOnly = errors.New("cron schedule supports exact minute and hour value only")
	ErrNotAStandardCronSpec                         = errors.New("not a standard cron spec(https://en.wikipedia.org/wiki/Cron)")
)

const (
	cronAnyRangeListCharacters = "*/-,"
	CronFieldSeparator         = " "
)

func validateCronSpec(spec string) error {
	if spec[0] != '@' {
		fields := strings.Split(spec, CronFieldSeparator)
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

func (s *CronScheduler) Contains(when time.Time) bool {
	whenInTimezone := when.In(s.timezone)
	lastStart := s.startSched.Previous(whenInTimezone)
	nextStart := s.startSched.Next(whenInTimezone)
	nextEnd := s.endSched.Next(whenInTimezone)
	// when in [lastStart, nextEnd) and nextStart > nextEnd
	return whenInTimezone.After(lastStart) && whenInTimezone.Before(nextEnd) && nextStart.After(nextEnd)
}
