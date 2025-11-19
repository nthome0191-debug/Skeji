package flows

import (
	maestro "skeji/internal/maestro/core"
	"skeji/pkg/config"
	"skeji/pkg/model"
	"time"
)

// requires: phone
// optional: city, label, start time, end time
func GetDailySchedule(ctx *maestro.MaestroContext) error {
	phone := ""
	cities := []string{}
	labels := []string{}
	start := time.Now()
	end := start.Add(24 * time.Hour)

	resp, err := ctx.Client.BusinessUnitClient.GetByPhone(phone, cities, labels, config.DefaultMaxBusinessUnitsPerAdminPhone, 0)
	if err != nil {
		return err
	}
	units, _, err := ctx.Client.BusinessUnitClient.DecodeBusinessUnits(resp)
	if err != nil {
		return err
	}
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
	return nil
}
