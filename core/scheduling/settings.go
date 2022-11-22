package scheduling

import (
	"errors"
	"fmt"
	"strings"
	"time"

	wingv1 "github.com/xscaling/wing/api/v1"

	jsonpatch "gopkg.in/evanphx/json-patch.v5"
)

var (
	ErrTimezoneNotFound            = errors.New("timezone not found")
	ErrSchedulePeriodNotFound      = errors.New("schedule period not found, `start` or `end` field not exists")
	ErrInvalidSchedulePeriodFormat = errors.New("invalid schedule period format, accepts `cron` or `date`")
	ErrStartEndSpecCanNotBeEqual   = errors.New("start and end spec can not be equal")
)

// GetScheduledSettingsRaw returns the raw settings of LAST hit schedule one
func GetScheduledSettingsRaw(when time.Time, settings wingv1.TargetSettings) (payload []byte, err error) {
	payload = make([]byte, len(settings.Default.Raw))
	copy(payload, settings.Default.Raw)

	var (
		hitScheduleSettingsPayload []byte
	)
	for _, schedule := range settings.Schedules {
		scheduler, err := GetScheduler(schedule)
		if err != nil {
			return nil, err
		}

		if scheduler.Contains(when) {
			copy(hitScheduleSettingsPayload, schedule.Settings.Raw)
		}
	}

	if hitScheduleSettingsPayload != nil {
		patchedPayload, err := jsonpatch.MergePatch(payload, hitScheduleSettingsPayload)
		if err != nil {
			return nil, err
		}
		payload = patchedPayload
	}
	return payload, nil
}

func GetScheduler(scheduleSettings wingv1.ScheduleTargetSettings) (Scheduler, error) {
	start, end, tz, err := getSchedulePeriod(scheduleSettings)
	if err != nil {
		return nil, err
	}
	var (
		scheduler Scheduler
	)
	// Easy-Predict
	switch len(strings.Split(start, cronFieldSeparator)) {
	case 5:
		scheduler, err = NewCronScheduler(tz, start, end)
	default:
		return nil, ErrInvalidSchedulePeriodFormat
	}
	return scheduler, err
}

func getSchedulePeriod(scheduleSettings wingv1.ScheduleTargetSettings) (start, end string, locale *time.Location, err error) {
	if scheduleSettings.Timezone == "" {
		return "", "", nil, ErrTimezoneNotFound
	}
	locale, err = time.LoadLocation(scheduleSettings.Timezone)
	if err != nil {
		return "", "", nil, err
	}
	start, end = scheduleSettings.Start, scheduleSettings.End
	if start == "" || end == "" {
		return "", "", nil, ErrSchedulePeriodNotFound
	}
	if start == end {
		return "", "", nil, ErrStartEndSpecCanNotBeEqual
	}
	return
}

// nolint
// Unused currently, reserve for validation webhook
func ValidateScheduleSettings(scheduleSettings []wingv1.ScheduleTargetSettings) error {
	for index, settings := range scheduleSettings {
		_, _, _, err := getSchedulePeriod(settings)
		if err != nil {
			return fmt.Errorf("%w: broken schedule settings(%d)", ErrSchedulePeriodNotFound, index)
		}
	}
	return nil
}
