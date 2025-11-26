package flows

import (
	"fmt"
	"skeji/internal/maestro/core"
	"skeji/pkg/config"
	"skeji/pkg/model"
	"sync"
	"time"
)

type DailyScheduleBusinessUnitScheduleBookingsParticipant struct {
	Name  string `json:"name"`
	Phone string `json:"phone"`
}

type DailyScheduleBusinessUnitScheduleBooking struct {
	Start        time.Time                                               `json:"start"`
	End          time.Time                                               `json:"end"`
	Label        string                                                  `json:"label"`
	Participants []*DailyScheduleBusinessUnitScheduleBookingsParticipant `json:"participants"`
}

type DailyScheduleBusinessUnitSchedule struct {
	Name     string                                      `json:"name"`
	City     string                                      `json:"city"`
	Address  string                                      `json:"address"`
	Bookings []*DailyScheduleBusinessUnitScheduleBooking `json:"bookings"`
}

type DailyScheduleBusinessUnits struct {
	Name      string                               `json:"name"`
	Labels    []string                             `json:"labels"`
	Schedules []*DailyScheduleBusinessUnitSchedule `json:"schedules"`
}

type DailySchedule struct {
	Units []*DailyScheduleBusinessUnits `json:"units"`
}

func GetDailySchedule(ctx *core.MaestroContext) error {
	phone := ctx.ExtractString("phone")
	if phone == "" {
		return nil
	}
	start, end := extractTimeFrame(ctx)
	units, err := fetchBusinessUnits(ctx, phone)
	if err != nil {
		return err
	}
	daily := buildDailySchedule(ctx, units, start, end)
	ctx.Output["result"] = daily
	return nil
}

func fetchBusinessUnits(ctx *core.MaestroContext, phone string) ([]*model.BusinessUnit, error) {
	cities := ctx.ExtractStringList("cities")
	labels := ctx.ExtractStringList("labels")
	resp, err := ctx.Client.BusinessUnitClient.GetByPhone(
		phone,
		cities,
		labels,
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

func buildDailySchedule(ctx *core.MaestroContext, units []*model.BusinessUnit, start, end time.Time) *DailySchedule {
	daily := &DailySchedule{
		Units: make([]*DailyScheduleBusinessUnits, len(units)),
	}

	var wg sync.WaitGroup

	for i, bu := range units {
		i, bu := i, bu
		wg.Add(1)

		core.RunWithRateLimitedConcurrency(func() {
			defer wg.Done()
			defer func() { _ = recover() }()
			daily.Units[i] = buildUnit(ctx, bu, start, end)
		})
	}

	wg.Wait()
	return daily
}

func buildUnit(ctx *core.MaestroContext, bu *model.BusinessUnit, start, end time.Time) *DailyScheduleBusinessUnits {
	unit := &DailyScheduleBusinessUnits{
		Name:   bu.Name,
		Labels: bu.Labels,
	}
	unit.Schedules = buildSchedules(ctx, bu, start, end)
	return unit
}

func buildSchedules(ctx *core.MaestroContext, bu *model.BusinessUnit, start, end time.Time) []*DailyScheduleBusinessUnitSchedule {
	var all []*DailyScheduleBusinessUnitSchedule
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

func buildCitySchedules(ctx *core.MaestroContext, buID string, scheds []*model.Schedule, start, end time.Time) []*DailyScheduleBusinessUnitSchedule {
	results := make([]*DailyScheduleBusinessUnitSchedule, len(scheds))
	var wg sync.WaitGroup
	for i, schedule := range scheds {
		i, schedule := i, schedule
		wg.Add(1)
		core.RunWithRateLimitedConcurrency(func() {
			defer wg.Done()
			defer func() { _ = recover() }()
			results[i] = buildSchedule(ctx, buID, schedule, start, end)
		})
	}
	wg.Wait()
	return results
}
func buildSchedule(ctx *core.MaestroContext, buID string, schedule *model.Schedule, start, end time.Time) *DailyScheduleBusinessUnitSchedule {
	sc := &DailyScheduleBusinessUnitSchedule{
		Name:    schedule.Name,
		City:    schedule.City,
		Address: schedule.Address,
	}
	sc.Bookings = buildBookings(ctx, buID, schedule.ID, start, end)
	return sc
}

func buildBookings(ctx *core.MaestroContext, buID, scheduleID string, start, end time.Time) []*DailyScheduleBusinessUnitScheduleBooking {
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
		return []*DailyScheduleBusinessUnitScheduleBooking{}
	}
	books, _, err := ctx.Client.BookingClient.DecodeBookings(resp)
	if err != nil {
		ctx.Logger.Warn(fmt.Sprintf("booking decode failed, err: %v\nresp: %v", err, resp))
		return []*DailyScheduleBusinessUnitScheduleBooking{}
	}
	if len(books) == 0 {
		return []*DailyScheduleBusinessUnitScheduleBooking{}
	}
	result := make([]*DailyScheduleBusinessUnitScheduleBooking, len(books))
	for i, book := range books {
		result[i] = buildBooking(book)
	}
	return result
}

func buildBooking(book *model.Booking) *DailyScheduleBusinessUnitScheduleBooking {
	b := &DailyScheduleBusinessUnitScheduleBooking{
		Start: book.StartTime,
		End:   book.EndTime,
		Label: book.ServiceLabel,
	}
	b.Participants = buildParticipants(book.Participants)
	return b
}

func buildParticipants(parts map[string]string) []*DailyScheduleBusinessUnitScheduleBookingsParticipant {
	if len(parts) == 0 {
		return []*DailyScheduleBusinessUnitScheduleBookingsParticipant{}
	}
	participants := make([]*DailyScheduleBusinessUnitScheduleBookingsParticipant, 0, len(parts))
	for name, phone := range parts {
		participants = append(participants, &DailyScheduleBusinessUnitScheduleBookingsParticipant{
			Name:  name,
			Phone: phone,
		})
	}

	return participants
}

func extractTimeFrame(ctx *core.MaestroContext) (time.Time, time.Time) {
	now := time.Now()

	start, err := ctx.ExtractTime("start")
	if err != nil {
		start = now
	}

	maxStart := now.Add(24 * time.Hour)
	minStart := now.Add(-24 * time.Hour)

	if start.After(maxStart) {
		start = maxStart
	}
	if start.Before(minStart) {
		start = minStart
	}

	end, err := ctx.ExtractTime("end")
	if err != nil {
		end = start.Add(10 * time.Hour)
	}

	if end.Before(start.Add(1 * time.Hour)) {
		end = start.Add(1 * time.Hour)
	}

	if end.After(start.Add(24 * time.Hour)) {
		end = start.Add(24 * time.Hour)
	}

	return start, end
}
