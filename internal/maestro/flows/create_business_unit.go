package flows

import (
	"fmt"
	"net/http"
	maestro "skeji/internal/maestro/core"
	"skeji/pkg/model"
)

func CreateBusinessUnit(ctx *maestro.MaestroContext) error {
	name := ctx.ExtractString("name")
	if maestro.IsMissing(name) {
		return maestro.MissingParamErr("name")
	}

	adminPhone := ctx.ExtractString("admin_phone")
	if maestro.IsMissing(adminPhone) {
		return maestro.MissingParamErr("admin_phone")
	}

	cities := ctx.ExtractStringList("cities")
	if len(cities) == 0 {
		return maestro.MissingParamErr("cities")
	}

	labels := ctx.ExtractStringList("labels")
	if len(labels) == 0 {
		return maestro.MissingParamErr("labels")
	}

	businessUnit := &model.BusinessUnit{
		Name:       name,
		AdminPhone: adminPhone,
		Cities:     cities,
		Labels:     labels,
	}

	timeZone := ctx.ExtractString("time_zone")
	if !maestro.IsMissing(timeZone) {
		businessUnit.TimeZone = timeZone
	}

	websiteURLs := ctx.ExtractStringList("website_urls")
	if len(websiteURLs) > 0 {
		businessUnit.WebsiteURLs = websiteURLs
	}

	maintainers := ctx.ExtractStringMap("maintainers")
	if len(maintainers) > 0 {
		businessUnit.Maintainers = maintainers
	}

	startOfDay := ctx.ExtractString("start_of_day")
	endOfDay := ctx.ExtractString("end_of_day")
	workingDays := ctx.ExtractStringList("working_days")
	scheduleTimeZone := ctx.ExtractString("schedule_time_zone")
	exceptions := ctx.ExtractStringList("exceptions")

	var defaultMeetingDurationMin int
	var hasDefaultMeetingDuration bool
	if val, exists := ctx.Input["default_meeting_duration_min"]; exists && val != nil {
		switch v := val.(type) {
		case int:
			defaultMeetingDurationMin = v
			hasDefaultMeetingDuration = true
		case int64:
			defaultMeetingDurationMin = int(v)
			hasDefaultMeetingDuration = true
		case float64:
			defaultMeetingDurationMin = int(v)
			hasDefaultMeetingDuration = true
		}
	}

	var defaultBreakDurationMin int
	var hasDefaultBreakDuration bool
	if val, exists := ctx.Input["default_break_duration_min"]; exists && val != nil {
		switch v := val.(type) {
		case int:
			defaultBreakDurationMin = v
			hasDefaultBreakDuration = true
		case int64:
			defaultBreakDurationMin = int(v)
			hasDefaultBreakDuration = true
		case float64:
			defaultBreakDurationMin = int(v)
			hasDefaultBreakDuration = true
		}
	}

	var maxParticipantsPerSlot int
	var hasMaxParticipants bool
	if val, exists := ctx.Input["max_participants_per_slot"]; exists && val != nil {
		switch v := val.(type) {
		case int:
			maxParticipantsPerSlot = v
			hasMaxParticipants = true
		case int64:
			maxParticipantsPerSlot = int(v)
			hasMaxParticipants = true
		case float64:
			maxParticipantsPerSlot = int(v)
			hasMaxParticipants = true
		}
	}
	fmt.Println("natali print 1")
	resp, err := ctx.Client.BusinessUnitClient.Create(businessUnit)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("failed to create business unit: %+v", resp.ToString())
	}
	fmt.Println("natali print 2")
	createdBU, err := ctx.Client.BusinessUnitClient.DecodeBusinessUnit(resp)
	if err != nil {
		return err
	}
	fmt.Println("natali print 3")
	createdSchedules := make([]*model.Schedule, 0, len(createdBU.Cities))

	for i, city := range createdBU.Cities {
		fmt.Printf("natali print 4 - %v\n", i)
		schedule := &model.Schedule{
			BusinessID: createdBU.ID,
			Name:       createdBU.Name + "_" + city,
			City:       city,
			Address:    city,
		}

		if !maestro.IsMissing(startOfDay) {
			schedule.StartOfDay = startOfDay
		}
		if !maestro.IsMissing(endOfDay) {
			schedule.EndOfDay = endOfDay
		}
		if len(workingDays) > 0 {
			schedule.WorkingDays = workingDays
		}
		if !maestro.IsMissing(scheduleTimeZone) {
			schedule.TimeZone = scheduleTimeZone
		}
		if hasDefaultMeetingDuration {
			schedule.DefaultMeetingDurationMin = defaultMeetingDurationMin
		}
		if hasDefaultBreakDuration {
			schedule.DefaultBreakDurationMin = defaultBreakDurationMin
		}
		if hasMaxParticipants {
			schedule.MaxParticipantsPerSlot = maxParticipantsPerSlot
		}
		if len(exceptions) > 0 {
			schedule.Exceptions = exceptions
		}

		schedResp, err := ctx.Client.ScheduleClient.Create(schedule)
		if err != nil {
			return fmt.Errorf("failed to create schedule for city %s: %v", city, err)
		}
		if schedResp.StatusCode != http.StatusCreated {
			return fmt.Errorf("failed to create schedule for city %s: %+v", city, schedResp.ToString())
		}

		createdSchedule, err := ctx.Client.ScheduleClient.DecodeSchedule(schedResp)
		if err != nil {
			return fmt.Errorf("failed to decode schedule for city %s: %v", city, err)
		}

		createdSchedules = append(createdSchedules, createdSchedule)
	}

	ctx.Output["business_unit"] = createdBU
	ctx.Output["schedules"] = createdSchedules
	return nil
}
