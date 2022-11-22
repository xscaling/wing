package scheduling

import (
	"testing"
	"time"

	wingv1 "github.com/xscaling/wing/api/v1"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func isSchedulePeriodContains(when time.Time, scheduleSettings wingv1.ScheduleTargetSettings) (bool, error) {
	scheduler, err := GetScheduler(scheduleSettings)
	if err != nil {
		return false, err
	}
	return scheduler.Contains(when), nil
}

func TestIsSchedulePeriodContains(t *testing.T) {
	const fixedTZ = "Asia/Shanghai"
	location, err := time.LoadLocation(fixedTZ)
	require.NoError(t, err)

	fixedDate, _ := time.ParseInLocation(schedulePeriodDateFormat, "2019-01-03 9:01", location)
	_, err = isSchedulePeriodContains(fixedDate, wingv1.ScheduleTargetSettings{
		Start: "0 0 * * *",
		End:   "0 1 * * *",
	})
	require.ErrorAs(t, err, &ErrTimezoneNotFound)

	_, err = isSchedulePeriodContains(fixedDate, wingv1.ScheduleTargetSettings{
		Timezone: fixedTZ,
		End:      "0 1 * * *",
	})
	require.ErrorAs(t, err, &ErrSchedulePeriodNotFound)

	_, err = isSchedulePeriodContains(fixedDate, wingv1.ScheduleTargetSettings{
		Timezone: fixedTZ,
		Start:    "0 1 * * *",
	})
	require.ErrorAs(t, err, &ErrSchedulePeriodNotFound)

	for _, testCase := range []struct {
		date       string
		start      string
		end        string
		causeError error
		contains   bool
	}{
		{
			date:     "2019-01-03 9:01",
			start:    "0 9 * * *",
			end:      "2 10 * * *",
			contains: true,
		},
		{
			date:     "2019-01-04 9:00",
			start:    "0 9 * * *",
			end:      "2 10 * * *",
			contains: true,
		},
		{
			date:     "2019-01-05 10:02",
			start:    "0 9 * * *",
			end:      "2 10 * * *",
			contains: false,
		},
		{
			date:     "2019-01-06 10:03",
			start:    "0 9 * * *",
			end:      "2 10 * * *",
			contains: false,
		},
		{
			date:     "2019-01-07 8:59",
			start:    "0 9 * * *",
			end:      "2 10 * * *",
			contains: false,
		},
		{
			date:     "2019-01-07 9:01",
			start:    "0 9 * * *",
			end:      "59 8 * * *",
			contains: true,
		},
		{
			date:     "2019-01-07 9:00",
			start:    "0 9 * * *",
			end:      "59 8 * * *",
			contains: true,
		},
		{
			date:     "2019-01-07 9:00",
			start:    "1 9 * * *",
			end:      "59 8 * * *",
			contains: false,
		},
		{
			date:     "2019-01-08 9:00",
			start:    "0 0 10 * *",
			end:      "0 0 11 * *",
			contains: false,
		},
		{
			date:     "2019-01-09 9:00",
			start:    "0 0 8 * *",
			end:      "0 0 10 * *",
			contains: true,
		},
		{
			// Whole weekend
			date:     "2021-12-04 9:00",
			start:    "0 0 * * 6",
			end:      "0 0 * * 1",
			contains: true,
		},
		{
			// Whole weekend
			date:     "2021-12-04 0:00",
			start:    "0 0 * * 6",
			end:      "0 0 * * 1",
			contains: true,
		},
		{
			// Whole weekend
			date:     "2021-12-06 0:00",
			start:    "0 0 * * 6",
			end:      "0 0 * * 1",
			contains: false,
		},
		{
			// Whole weekend
			date:     "2021-12-05 23:59",
			start:    "0 0 * * 6",
			end:      "0 0 * * 1",
			contains: true,
		},
		{
			date:     "2021-12-05 23:59",
			start:    "1 0 * * 6",
			end:      "0 0 * * 1",
			contains: true,
		},
		{
			date:     "2021-12-02 18:59",
			start:    "0 0 * * 3",
			end:      "0 23 * * 3",
			contains: false,
		},
		{
			date:       "2021-12-02 18:59",
			start:      "0 0 * * 3",
			end:        "2021-12-03 19:00",
			causeError: ErrInvalidSchedulePeriodFormat,
		},
		{
			date:       "2021-12-02 18:59",
			start:      "2021-12-03 19:00",
			end:        "0 0 * * 3",
			causeError: ErrInvalidSchedulePeriodFormat,
		},
		{
			date:       "2021-12-05 0:00",
			start:      "0 0 * * 3",
			end:        "0 0 * * 3",
			causeError: ErrStartEndSpecCanNotBeEqual,
		},
		{
			date:       "2019-01-01 9:01",
			start:      "* 9 * * *",
			end:        "2 10 * * *",
			contains:   false,
			causeError: ErrCronScheduleSupportsExactMinuteHourValueOnly,
		},
		{
			date:       "2019-01-02 9:01",
			start:      "* * * * *",
			end:        "2 10 * * *",
			contains:   false,
			causeError: ErrCronScheduleSupportsExactMinuteHourValueOnly,
		},
		{
			date:       "2021-12-05 23:59",
			start:      "0 0 * * 6",
			end:        "0-1 0 * * 1",
			contains:   false,
			causeError: ErrCronScheduleSupportsExactMinuteHourValueOnly,
		},
		{
			date:       "2021-12-05 23:59",
			start:      "0-1 0 * * 6",
			end:        "0 0 * * 1",
			contains:   false,
			causeError: ErrCronScheduleSupportsExactMinuteHourValueOnly,
		},
		{
			date:       "2021-12-05 23:59",
			start:      "0 0-1 * * 6",
			end:        "0 0 * * 1",
			contains:   false,
			causeError: ErrCronScheduleSupportsExactMinuteHourValueOnly,
		},
		{
			date:       "2021-12-05 23:59",
			start:      "0 0 * * 6",
			end:        "0 0-1 * * 1",
			contains:   false,
			causeError: ErrCronScheduleSupportsExactMinuteHourValueOnly,
		},
		{
			date:       "2021-12-05 23:59",
			start:      "0 0 * * 6 *",
			end:        "0 0 * * 1",
			contains:   false,
			causeError: ErrNotAStandardCronSpec,
		},
		{
			date:       "2021-12-05 23:59",
			start:      "0 0 * * 6",
			end:        "0 0 * * 1 *",
			contains:   false,
			causeError: ErrNotAStandardCronSpec,
		},
	} {
		date, _err := time.ParseInLocation(schedulePeriodDateFormat, testCase.date, location)
		require.NoError(t, _err)
		contains, err := isSchedulePeriodContains(date, wingv1.ScheduleTargetSettings{
			Timezone: fixedTZ,
			Start:    testCase.start,
			End:      testCase.end,
		})
		assert.Equal(t, testCase.causeError != nil, err != nil,
			"unexpected error when: %s, start: %s, end: %s, expect %v got %v",
			testCase.date, testCase.start, testCase.end, testCase.causeError, err)
		if testCase.causeError != nil {
			require.ErrorAs(t, err, &testCase.causeError)
		} else {
			require.NoError(t, err)
			// Check contains
			require.Equal(t, testCase.contains, contains, "date: %s, start: %s, end: %s", testCase.date, testCase.start, testCase.end)
		}
	}
}

func BenchmarkIsSchedulePeriodContainsWithCron(b *testing.B) {
	const fixedTZ = "Asia/Shanghai"
	when := time.Now()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = isSchedulePeriodContains(when, wingv1.ScheduleTargetSettings{
			Timezone: fixedTZ,
			Start:    "0 0 * * *",
			End:      "0 0 * * *",
		})
	}
}

func BenchmarkIsSchedulePeriodContainsWithDate(b *testing.B) {
	const fixedTZ = "Asia/Shanghai"
	when := time.Now()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = isSchedulePeriodContains(when, wingv1.ScheduleTargetSettings{
			Timezone: fixedTZ,
			Start:    "2021-01-01 10:00",
			End:      "2021-01-01 20:00",
		})
	}
}
