package flows

import (
	"fmt"
	"net/http"
	maestro "skeji/internal/maestro/core"
	"skeji/internal/maestro/types"
	"skeji/pkg/model"
)

func CreateBusinessUnit(ctx *maestro.MaestroContext) error {
	input, err := types.FromMapCreateBusinessUnit(ctx.Input)
	if err != nil {
		return fmt.Errorf("invalid input: %w", err)
	}

	businessUnit := &model.BusinessUnit{
		Name:       input.Name,
		AdminPhone: input.AdminPhone,
		Cities:     input.Cities,
		Labels:     input.Labels,
	}

	if input.TimeZone != "" {
		businessUnit.TimeZone = input.TimeZone
	}
	if len(input.WebsiteURLs) > 0 {
		businessUnit.WebsiteURLs = input.WebsiteURLs
	}
	if len(input.Maintainers) > 0 {
		businessUnit.Maintainers = input.Maintainers
	}

	resp, err := ctx.Client.BusinessUnitClient.Create(businessUnit)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("failed to create business unit: %+v", resp.ToString())
	}
	createdBU, err := ctx.Client.BusinessUnitClient.DecodeBusinessUnit(resp)
	if err != nil {
		return err
	}

	for _, city := range createdBU.Cities {
		schedule := &model.Schedule{
			BusinessID: createdBU.ID,
			Name:       createdBU.Name + "_" + city,
			City:       city,
			Address:    city,
			TimeZone:   createdBU.TimeZone,
		}

		if input.StartOfDay != nil {
			schedule.StartOfDay = *input.StartOfDay
		}
		if input.EndOfDay != nil {
			schedule.EndOfDay = *input.EndOfDay
		}
		if len(input.WorkingDays) > 0 {
			schedule.WorkingDays = input.WorkingDays
		}
		if input.ScheduleTimeZone != nil {
			schedule.TimeZone = *input.ScheduleTimeZone
		}
		if input.DefaultMeetingDurationMin != nil {
			schedule.DefaultMeetingDurationMin = *input.DefaultMeetingDurationMin
		}
		if input.DefaultBreakDurationMin != nil {
			schedule.DefaultBreakDurationMin = *input.DefaultBreakDurationMin
		}
		if input.MaxParticipantsPerSlot != nil {
			schedule.MaxParticipantsPerSlot = *input.MaxParticipantsPerSlot
		}

		schedResp, err := ctx.Client.ScheduleClient.Create(schedule)
		if err != nil {
			return fmt.Errorf("failed to create schedule for city %s: %v", city, err)
		}
		if schedResp.StatusCode != http.StatusCreated {
			return fmt.Errorf("failed to create schedule for city %s: %+v", city, schedResp.ToString())
		}

		_, err = ctx.Client.ScheduleClient.DecodeSchedule(schedResp)
		if err != nil {
			return fmt.Errorf("failed to decode schedule for city %s: %v", city, err)
		}

		if !input.SchedulePerCity {
			break
		}
	}
	return nil
}
