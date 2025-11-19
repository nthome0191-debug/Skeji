package flows

import (
	maestro "skeji/internal/maestro/core"
	"skeji/pkg/config"
	"skeji/pkg/model"
	"time"
)

const (
	PHONE                      = "phone"
	START_TIME                 = "start_time"
	END_TIME                   = "end_time"
	BUSINESS_UNITS             = "business_units"
	BU_ID_TO_CITY_TO_SCHEDULES = "bu_id_to_city_to_schedules"

	BU_ID_TO_CITY_TO_SCHEDULE_ID_TO_BOOKINGS = "bu_id_to_city_to_schedule_id_to_bookings"

	MaxBookingsPerScheduleFetch = 30
)

func ListPhoneRelatedBusinessUnits(ctx *maestro.MaestroContext) error {
	phone := ctx.Input[PHONE].(string)
	resp, err := ctx.Client.BusinessUnitClient.GetByPhone(phone, config.DefaultMaxBusinessUnitsPerAdminPhone, 0)
	if err != nil {
		return err
	}
	units, _, err := ctx.Client.BusinessUnitClient.DecodeBusinessUnits(resp)
	if err != nil {
		return err
	}
	ctx.Process[BUSINESS_UNITS] = units
	return nil
}

func ListBusinessUnitsRelatedSchedules(ctx *maestro.MaestroContext) error {
	units := ctx.Process[BUSINESS_UNITS].([]*model.BusinessUnit)
	schedules := map[string]map[string][]*model.Schedule{}
	for _, unit := range units {
		schedules[unit.ID] = map[string][]*model.Schedule{}
		for _, city := range unit.Cities {
			resp, err := ctx.Client.ScheduleClient.Search(unit.ID, city, config.DefaultMaxSchedulesPerBusinessUnits, 0)
			if err != nil {
				return err
			}
			scheds, _, err := ctx.Client.ScheduleClient.DecodeSchedules(resp)
			if err != nil {
				return err
			}
			schedules[unit.ID][city] = scheds
		}
	}
	ctx.Process[BU_ID_TO_CITY_TO_SCHEDULES] = schedules
	return nil
}

func ListBusinessUnitsRelatedCityRelatedSchedulesRelatedBookings(ctx *maestro.MaestroContext) error {
	now := time.Now()

	start := now
	if raw, ok := ctx.Input[START_TIME]; ok {
		startStr, _ := raw.(string)
		if parsed, err := time.Parse(time.RFC3339, startStr); err == nil {
			if parsed.Before(now.Add(-24 * time.Hour)) {
				start = now.Add(-24 * time.Hour)
			} else {
				start = parsed
			}
		}
	}

	end := start.Add(24 * time.Hour)
	if raw, ok := ctx.Input[END_TIME]; ok {
		endStr, _ := raw.(string)
		if parsed, err := time.Parse(time.RFC3339, endStr); err == nil {
			end = parsed
		}
	}

	ctx.Process["start_time"] = start
	ctx.Process["end_time"] = end

	schedules := ctx.Process[BU_ID_TO_CITY_TO_SCHEDULES].(map[string]map[string][]*model.Schedule)
	bookings := map[string]map[string]map[string][]*model.Booking{}

	for buid, citiesToSchedules := range schedules {
		bookings[buid] = map[string]map[string][]*model.Booking{}
		for city, schedules := range citiesToSchedules {
			bookings[buid][city] = map[string][]*model.Booking{}
			for _, schedule := range schedules {
				resp, err := ctx.Client.BookingClient.Search(buid, schedule.ID, start.Format(time.RFC3339), end.Format(time.RFC3339), 30, 0)
				if err != nil {
					return err
				}
				bs, _, err := ctx.Client.BookingClient.DecodeBookings(resp)
				if err != nil {
					return err
				}
				bookings[buid][city][schedule.ID] = bs
			}
		}
	}
	ctx.Process["bu_id_to_city_to_schedule_id_to_bookings"] = bookings
	return nil
}

func OrganizeOutput(ctx *maestro.MaestroContext) error {
	return nil
}
