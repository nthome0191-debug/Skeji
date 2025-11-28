package flows

import (
	"fmt"
	"skeji/internal/maestro/core"
	"skeji/internal/maestro/types"
	"skeji/pkg/config"
	"skeji/pkg/model"
	"sync"
	"time"
)

func GetDailySchedule(ctx *core.MaestroContext) error {
	input, err := types.FromMapGetDailySchedule(ctx.Input)
	if err != nil {
		return fmt.Errorf("invalid input: %w", err)
	}

	start, end := extractTimeFrame(input)

	units, err := fetchBusinessUnits(ctx, input)
	if err != nil {
		return err
	}

	daily := buildDailySchedule(ctx, units, start, end)
	ctx.Output["result"] = daily
	return nil
}

func fetchBusinessUnits(ctx *core.MaestroContext, input *types.GetDailyScheduleInput) ([]*model.BusinessUnit, error) {
	resp, err := ctx.Client.BusinessUnitClient.GetByPhone(
		input.Phone,
		input.Cities,
		input.Labels,
		config.DefaultMaxBusinessUnitsPerAdminPhone,
		0,
	)
	if err != nil {
		return nil, err
	}
	units, _, err := ctx.Client.BusinessUnitClient.DecodeBusinessUnits(resp)
	if err != nil {
		return nil, err
	}
	return units, nil
}

func buildDailySchedule(ctx *core.MaestroContext, units []*model.BusinessUnit, start, end time.Time) *types.DailySchedule {
	daily := &types.DailySchedule{
		Units: make([]*types.DailyScheduleBusinessUnit, len(units)),
	}

	var wg sync.WaitGroup

	for i, bu := range units {
		i, bu := i, bu
		wg.Add(1)

		core.RunWithRateLimitedConcurrency(func() {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					ctx.Logger.Error(fmt.Sprintf("panic while building unit for business %s: %v", bu.ID, r))
					// Leave the unit as nil - caller will see incomplete results
				}
			}()
			daily.Units[i] = buildUnit(ctx, bu, start, end)
		})
	}

	wg.Wait()
	return daily
}

func buildUnit(ctx *core.MaestroContext, bu *model.BusinessUnit, start, end time.Time) *types.DailyScheduleBusinessUnit {
	unit := &types.DailyScheduleBusinessUnit{
		Name:   bu.Name,
		Labels: bu.Labels,
	}
	unit.Schedules = buildSchedules(ctx, bu, start, end)
	return unit
}

func buildSchedules(ctx *core.MaestroContext, bu *model.BusinessUnit, start, end time.Time) []*types.DailyScheduleSchedule {
	var all []*types.DailyScheduleSchedule
	for _, city := range bu.Cities {
		resp, err := ctx.Client.ScheduleClient.Search(
			bu.ID,
			city,
			config.DefaultMaxSchedulesPerBusinessUnits,
			0,
		)
		if err != nil {
			ctx.Logger.Warn(fmt.Sprintf("schedules search failed, err: %v", err))
			continue
		}
		scheds, _, err := ctx.Client.ScheduleClient.DecodeSchedules(resp)
		if err != nil {
			ctx.Logger.Warn(fmt.Sprintf("schedules decode failed, err: %v\nresp: %v", err, resp))
			continue
		}
		citySchedules := buildCitySchedules(ctx, bu.ID, scheds, start, end)
		if len(citySchedules) > 0 {
			all = append(all, citySchedules...)
		}
	}

	return all
}

func buildCitySchedules(ctx *core.MaestroContext, buID string, scheds []*model.Schedule, start, end time.Time) []*types.DailyScheduleSchedule {
	results := make([]*types.DailyScheduleSchedule, len(scheds))
	var wg sync.WaitGroup
	for i, schedule := range scheds {
		i, schedule := i, schedule
		wg.Add(1)
		core.RunWithRateLimitedConcurrency(func() {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					ctx.Logger.Error(fmt.Sprintf("panic while building schedule %s for business %s: %v", schedule.ID, buID, r))
					// Leave the schedule as nil - caller will see incomplete results
				}
			}()
			results[i] = buildSchedule(ctx, buID, schedule, start, end)
		})
	}
	wg.Wait()
	return results
}

func buildSchedule(ctx *core.MaestroContext, buID string, schedule *model.Schedule, start, end time.Time) *types.DailyScheduleSchedule {
	sc := &types.DailyScheduleSchedule{
		Name:    schedule.Name,
		City:    schedule.City,
		Address: schedule.Address,
	}
	sc.Bookings = buildBookings(ctx, buID, schedule.ID, start, end)
	return sc
}

func buildBookings(ctx *core.MaestroContext, buID, scheduleID string, start, end time.Time) []*types.DailyScheduleBooking {
	resp, err := ctx.Client.BookingClient.Search(
		buID,
		scheduleID,
		start.Format(time.RFC3339),
		end.Format(time.RFC3339),
		config.DefaultMaxBookingsPerView,
		0,
	)
	if err != nil {
		ctx.Logger.Warn(fmt.Sprintf("booking search failed, err: %v", err))
		return []*types.DailyScheduleBooking{}
	}
	books, _, err := ctx.Client.BookingClient.DecodeBookings(resp)
	if err != nil {
		ctx.Logger.Warn(fmt.Sprintf("booking decode failed, err: %v\nresp: %v", err, resp))
		return []*types.DailyScheduleBooking{}
	}
	if len(books) == 0 {
		return []*types.DailyScheduleBooking{}
	}
	result := make([]*types.DailyScheduleBooking, len(books))
	for i, book := range books {
		result[i] = buildBooking(book)
	}
	return result
}

func buildBooking(book *model.Booking) *types.DailyScheduleBooking {
	b := &types.DailyScheduleBooking{
		Start: book.StartTime,
		End:   book.EndTime,
		Label: book.ServiceLabel,
	}
	b.Participants = buildParticipants(book.Participants)
	return b
}

func buildParticipants(parts map[string]string) []*types.DailyScheduleParticipant {
	if len(parts) == 0 {
		return []*types.DailyScheduleParticipant{}
	}
	participants := make([]*types.DailyScheduleParticipant, 0, len(parts))
	for name, phone := range parts {
		participants = append(participants, &types.DailyScheduleParticipant{
			Name:  name,
			Phone: phone,
		})
	}

	return participants
}

func extractTimeFrame(input *types.GetDailyScheduleInput) (time.Time, time.Time) {
	now := time.Now()
	var start time.Time
	if input.Start != nil {
		start = *input.Start
	} else {
		start = now
	}

	maxStart := now.Add(24 * time.Hour)
	minStart := now

	if start.After(maxStart) {
		start = maxStart
	}
	if start.Before(minStart) {
		start = minStart
	}

	var end time.Time
	if input.End != nil {
		end = *input.End
	} else {
		end = start.Add(36 * time.Hour)
	}

	if end.Before(start.Add(1 * time.Hour)) {
		end = start.Add(1 * time.Hour)
	}

	if end.After(start.Add(36 * time.Hour)) {
		end = start.Add(36 * time.Hour)
	}

	return start, end
}
