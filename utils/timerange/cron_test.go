package timerange

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func mustGetTime(date string) time.Time {
	when, err := time.Parse(SchedulePeriodDateFormat, date)
	if err != nil {
		panic(err)
	}
	return when
}

func TestParseCronScheduleSpec(t *testing.T) {
	for _, testCase := range []struct {
		when                string
		spec                string
		nextTimestampOffset int64
		causeError          error
	}{
		{
			when:                "2019-01-01 9:01",
			spec:                "0 9 * * *",
			nextTimestampOffset: int64((time.Hour*24 - time.Minute).Seconds()),
		},
		{
			when:                "2019-01-01 9:01",
			spec:                "1 9 * * *",
			nextTimestampOffset: int64((time.Hour * 24).Seconds()),
		},
		{
			when:                "2019-01-01 9:01",
			spec:                "@hourly",
			nextTimestampOffset: int64((time.Hour - time.Minute).Seconds()),
		},
		// error cases
		{
			when:       "2019-01-01 9:01",
			spec:       "1 2 * *",
			causeError: ErrNotAStandardCronSpec,
		},
		{
			when:       "2019-01-01 9:01",
			spec:       "1-2 2 * *",
			causeError: ErrCronScheduleSupportsExactMinuteHourValueOnly,
		},
		{
			when:       "2019-01-01 9:01",
			spec:       "1,2,3 2 * *",
			causeError: ErrCronScheduleSupportsExactMinuteHourValueOnly,
		},
	} {
		when := mustGetTime(testCase.when)
		schedule, err := parseCronScheduleSpec(testCase.spec)
		require.Equal(t, testCase.causeError != nil, err != nil)
		if testCase.causeError != nil {
			require.ErrorAs(t, err, &testCase.causeError, "when: %s, spec: %s", testCase.when, testCase.spec)
		} else {
			require.NoError(t, err)
			require.Equal(t, testCase.nextTimestampOffset+when.Unix(), schedule.Next(when).Unix(),
				"when: %s, spec: %s", testCase.when, testCase.spec)
		}
	}
}

func TestCronScheduleContains(t *testing.T) {
	for _, c := range []struct {
		description string
		start       string
		end         string
		expected    map[string]bool
	}{
		{
			description: "Every day between 9:00 and 10:00",
			start:       "0 9 * * *",
			end:         "0 10 * * *",
			expected: map[string]bool{
				"2024-08-15 08:59": false,
				"2024-08-15 09:00": true,
				"2024-08-15 09:01": true,
				"2024-08-15 09:59": true,
				"2024-08-15 10:00": false,
				"2024-08-15 22:00": false,
			},
		},
		{
			description: "Every day between 10:00 and 09:00, crossing the midnight",
			start:       "0 10 * * *",
			end:         "0 9 * * *",
			expected: map[string]bool{
				"2024-08-15 09:59": false,
				"2024-08-15 10:01": true,
				"2024-08-15 23:59": true,
				"2024-08-16 00:00": true,
				"2024-08-16 00:01": true,
				"2024-08-16 08:59": true,
				"2024-08-16 09:00": false,
				"2024-08-16 09:01": false,
			},
		},
		{
			description: "10th,20th,30th month day between 9:00 and 10:00",
			start:       "0 9 10,20,30 * *",
			end:         "0 10 10,20,30 * *",
			expected: map[string]bool{
				"2024-08-01 00:00": false,
				"2024-08-09 23:59": false,
				"2024-08-10 00:00": false,
				"2024-08-10 08:59": false,
				"2024-08-10 09:00": true,
				"2024-08-10 09:30": true,
				"2024-08-10 09:59": true,
				"2024-08-10 10:00": false,
				"2024-08-10 23:59": false,
				"2024-08-11 00:00": false,
				"2024-08-19 23:59": false,
				"2024-08-20 00:00": false,
				"2024-08-20 08:59": false,
				"2024-08-20 09:00": true,
				"2024-08-20 09:30": true,
				"2024-08-20 09:59": true,
				"2024-08-20 10:00": false,
				"2024-08-20 23:59": false,
				"2024-08-21 00:00": false,
				"2024-08-29 23:59": false,
				"2024-08-30 00:00": false,
				"2024-08-30 08:59": false,
				"2024-08-30 09:00": true,
				"2024-08-30 09:30": true,
				"2024-08-30 09:59": true,
				"2024-08-30 10:00": false,
				"2024-08-30 23:59": false,
				"2024-08-31 00:00": false,
				"2024-08-31 23:59": false,
			},
		},
		{
			description: "Worktime between 9:00 and 19:00",
			start:       "0 9 * * 1-5",
			end:         "0 19 * * 1-5",
			expected: map[string]bool{
				"2024-08-11 23:59": false, // Last minute of a Sunday
				"2024-08-12 08:59": false, // First minute of a Monday before-work breaktime
				"2024-08-12 09:00": true,  // First minute of a Monday worktime
				"2024-08-12 09:01": true,
				"2024-08-12 14:00": true,
				"2024-08-12 18:59": true,  // Last minute of a Monday worktime
				"2024-08-12 19:00": false, // First minute of a Monday after-work breaktime
				"2024-08-12 19:01": false,
				"2024-08-16 08:59": false, // First minute of a Friday before-work breaktime
				"2024-08-16 09:00": true,  // First minute of a Friday worktime
				"2024-08-16 18:59": true,  // Last minute of a Friday worktime
				"2024-08-16 19:00": false, // First minute of a Friday after-work breaktime
				"2024-08-17 09:00": false, // Any minute of a weekend
				"2024-08-17 18:59": false,
				"2024-08-18 18:59": false,
			},
		},
		{
			description: "Summer vocation for most Chinese students",
			start:       "0 0 1 7 *",
			end:         "0 0 1 9 *",
			expected: map[string]bool{
				"2024-06-30 23:59": false, // Last minute of June
				"2024-07-01 00:00": true,  // First minute of July
				"2024-07-01 00:01": true,  // Any minute of summer vocation
				"2024-08-15 15:11": true,
				"2024-08-31 23:59": true,  // Last minute of August
				"2024-09-01 00:00": false, // Buddy, it's time for school
				"2024-02-15 00:00": false, // To check execution duration with the farthest minute
			},
		},
		{
			description: "Every Thursday between 9:00 and 10:00, no influence on date August 15th",
			start:       "0 9 15 8 4",
			end:         "0 10 15 8 4",
			expected: map[string]bool{
				"2024-08-08 08:59": false, // Thursday before
				"2024-08-08 09:00": true,
				"2024-08-08 09:59": true,
				"2024-08-08 10:00": false,
				"2023-08-14 08:59": false, // Wednesday
				"2023-08-14 09:00": false,
				"2024-08-15 08:59": false, // Thursday exactly
				"2024-08-15 09:00": true,
				"2024-08-15 09:59": true,
				"2024-08-15 10:00": false,
				"2023-08-16 08:59": false, // Friday
				"2023-08-16 09:00": false,
				"2024-08-22 08:59": false, // Thursday after
				"2024-08-22 09:00": true,
				"2024-08-22 09:59": true,
				"2024-08-22 10:00": false,
			},
		},
		{
			description: "Schedule range are both @hourly, which matches nothing",
			start:       "@hourly",
			end:         "@hourly",
			expected: map[string]bool{
				"2019-01-01 08:59": false,
				"2024-08-15 09:00": false,
			},
		},
		{
			description: "Schedule across over weekend",
			start:       "0 0 * * 6",
			end:         "0 0 * * 1",
			expected: map[string]bool{
				"2024-07-05 23:59": false, // Last minute of a Friday
				"2024-07-06 00:00": true,  // First minute of a Saturday
				"2024-07-06 00:01": true,  // Any minute of a weekend
				"2024-07-07 23:59": true,  // Last minute of a Sunday
				"2024-07-08 00:00": false, // First minute of a Monday
			},
		},
		{
			description: "Schedule across over weekend with range",
			start:       "0 0 * * 6,1",
			end:         "59 23 * * 6,1",
			expected: map[string]bool{
				"2024-08-17 00:00": true,  // First minute of a Saturday
				"2024-08-17 23:00": true,  // Any minute of a Saturday
				"2024-08-17 23:59": false, // Last minute of a Saturday
				"2024-08-18 00:00": false, // First minute of a Sunday
				"2024-08-18 23:01": false, // Any minute of a Sunday
				"2024-08-18 23:59": false, // Last minute of a Sunday
				"2024-08-19 00:00": true,  // First minute of a Monday
				"2024-08-19 08:00": true,  // Any minute of a Monday
				"2024-08-19 23:59": false, // Last minute of a Monday
				"2024-08-20 00:00": false, // First minute of a Tuesday
			},
		},
		{
			description: "Sunday issue",
			start:       "0 0 * * 0", // Zero means Sunday
			end:         "59 23 * * 0",
			expected: map[string]bool{
				"2024-08-17 23:00": false, // Any minute of a Saturday
				"2024-08-17 23:59": false, // Last minute of a Saturday
				"2024-08-18 00:00": true,  // First minute of a Sunday
				"2024-08-18 23:01": true,  // Any minute of a Sunday
				"2024-08-18 23:59": false, // Last minute of a Sunday
				"2024-08-19 00:00": false, // First minute of a Monday
				"2024-08-19 23:59": false, // Last minute of a Monday
			},
		},
	} {
		s, err := NewCronScheduler(time.UTC, c.start, c.end)
		require.NoError(t, err)
		for when, expected := range c.expected {
			t.Run(c.description+": "+when, func(t *testing.T) {
				gotIn := s.Contains(mustGetTime(when))
				require.Equal(t, expected, gotIn,
					"[%s] start: %s, end: %s, when: %s", c.description, c.start, c.end, when)
			})
		}
	}
}
