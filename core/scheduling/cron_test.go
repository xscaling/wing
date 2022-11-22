package scheduling

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestParseCronScheduleSpec(t *testing.T) {
	mustGetTime := func(date string) time.Time {
		when, err := time.Parse(schedulePeriodDateFormat, date)
		if err != nil {
			panic(err)
		}
		return when
	}
	for _, testCase := range []struct {
		when                time.Time
		spec                string
		nextTimestampOffset int64
		causeError          error
	}{
		{
			when:                mustGetTime("2019-01-01 9:01"),
			spec:                "0 9 * * *",
			nextTimestampOffset: int64((time.Hour*24 - time.Minute).Seconds()),
		},
		{
			when:                mustGetTime("2019-01-01 9:01"),
			spec:                "1 9 * * *",
			nextTimestampOffset: int64((time.Hour * 24).Seconds()),
		},
		{
			when:                mustGetTime("2019-01-01 9:01"),
			spec:                "@hourly",
			nextTimestampOffset: int64((time.Hour - time.Minute).Seconds()),
		},
		// error cases
		{
			when:       mustGetTime("2019-01-01 9:01"),
			spec:       "1 2 * *",
			causeError: ErrNotAStandardCronSpec,
		},
		{
			when:       mustGetTime("2019-01-01 9:01"),
			spec:       "1-2 2 * *",
			causeError: ErrCronScheduleSupportsExactMinuteHourValueOnly,
		},
		{
			when:       mustGetTime("2019-01-01 9:01"),
			spec:       "1,2,3 2 * *",
			causeError: ErrCronScheduleSupportsExactMinuteHourValueOnly,
		},
	} {
		schedule, err := parseCronScheduleSpec(testCase.spec)
		require.Equal(t, testCase.causeError != nil, err != nil)
		if testCase.causeError != nil {
			require.ErrorAs(t, err, &testCase.causeError, "when: %s, spec: %s", testCase.when, testCase.spec)
		} else {
			require.NoError(t, err)
			require.Equal(t, testCase.nextTimestampOffset+testCase.when.Unix(), schedule.Next(testCase.when).Unix(),
				"when: %s, spec: %s", testCase.when, testCase.spec)
		}
	}
}
