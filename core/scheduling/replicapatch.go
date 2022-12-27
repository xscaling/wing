package scheduling

import (
	"strings"
	"time"

	wingv1 "github.com/xscaling/wing/api/v1"
	"github.com/xscaling/wing/utils/timerange"
)

func GetReplicaPatch(when time.Time, patches wingv1.ReplicaPatches) (*wingv1.ReplicaPatch, error) {
	var (
		start string
		end   string
	)
	for _, patch := range patches {
		if patch.Timezone == "" {
			return nil, ErrTimezoneNotFound
		}
		timezone, err := time.LoadLocation(patch.Timezone)
		if err != nil {
			return nil, err
		}
		start, end = patch.Start, patch.End
		if start == "" || end == "" {
			return nil, ErrSchedulePeriodNotFound
		}
		if start == end {
			return nil, ErrStartEndSpecCanNotBeEqual
		}

		var (
			scheduler timerange.Scheduler
		)
		// Easy-Predict
		switch len(strings.Split(start, timerange.CronFieldSeparator)) {
		case 2:
			scheduler, err = timerange.NewDateScheduler(timezone, start, end)
		case 5:
			scheduler, err = timerange.NewCronScheduler(timezone, start, end)
		default:
			return nil, timerange.ErrInvalidSchedulePeriodFormat
		}

		if err != nil {
			return nil, err
		}
		if scheduler.Contains(when) {
			return &patch, nil
		}
	}
	return nil, nil
}
