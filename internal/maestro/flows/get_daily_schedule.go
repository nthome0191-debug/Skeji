package flows

import (
	maestro "skeji/internal/maestro/core"
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

func GetDailySchedule(ctx *maestro.MaestroContext) error {
	phone := ctx.ExtractString("phone")
	if phone == "" {
		return nil
	}

	cities := ctx.ExtractStringList("cities")
	labels := ctx.ExtractStringList("labels")
	start, end := fetchAndApplyTimeFrameForView(ctx)

	maestro.ReqAcquire()
	resp, err := ctx.Client.BusinessUnitClient.GetByPhone(
		phone,
		cities,
		labels,
		config.DefaultMaxBusinessUnitsPerAdminPhone,
		0,
	)
	maestro.ReqRelease()
	if err != nil {
		return err
	}

	units, _, err := ctx.Client.BusinessUnitClient.DecodeBusinessUnits(resp)
	if err != nil {
		return err
	}

	dailySchedule := &DailySchedule{
		Units: make([]*DailyScheduleBusinessUnits, len(units)),
	}

	if err := fillBusinessUnits(ctx, dailySchedule, units, start, end); err != nil {
		return err
	}

	ctx.Output["daily_schedule"] = dailySchedule
	return nil
}

func fillBusinessUnits(
	ctx *maestro.MaestroContext,
	dailySchedule *DailySchedule,
	units []*model.BusinessUnit,
	start, end time.Time,
) error {
	var wg sync.WaitGroup
	var errMu sync.Mutex
	var firstErr error
	for i, unit := range units {
		wg.Add(1)
		go func() {
			defer wg.Done()
			bu := &DailyScheduleBusinessUnits{
				Name:      unit.Name,
				Labels:    unit.Labels,
				Schedules: []*DailyScheduleBusinessUnitSchedule{},
			}
			var cityWg sync.WaitGroup
			var cityMu sync.Mutex
			for _, city := range unit.Cities {
				cityWg.Add(1)
				go func() {
					defer cityWg.Done()
					maestro.ReqAcquire()
					resp, err := ctx.Client.ScheduleClient.Search(
						unit.ID,
						city,
						config.DefaultMaxSchedulesPerBusinessUnits,
						0,
					)
					maestro.ReqRelease()
					if err != nil {
						errMu.Lock()
						if firstErr == nil {
							firstErr = err
						}
						errMu.Unlock()
						return
					}
					scheds, _, err := ctx.Client.ScheduleClient.DecodeSchedules(resp)
					if err != nil {
						errMu.Lock()
						if firstErr == nil {
							firstErr = err
						}
						errMu.Unlock()
						return
					}
					schedules := make([]*DailyScheduleBusinessUnitSchedule, len(scheds))
					if err := fillSchedules(ctx, schedules, scheds, unit.ID, start, end); err != nil {
						errMu.Lock()
						if firstErr == nil {
							firstErr = err
						}
						errMu.Unlock()
						return
					}
					cityMu.Lock()
					bu.Schedules = append(bu.Schedules, schedules...)
					cityMu.Unlock()
				}()
			}
			cityWg.Wait()
			dailySchedule.Units[i] = bu
		}()
	}
	wg.Wait()
	return firstErr
}

func fillSchedules(
	ctx *maestro.MaestroContext,
	schedules []*DailyScheduleBusinessUnitSchedule,
	scheds []*model.Schedule,
	unitID string,
	start, end time.Time,
) error {
	var wg sync.WaitGroup
	var errMu sync.Mutex
	var firstErr error

	for j, schedule := range scheds {
		j, schedule := j, schedule

		wg.Add(1)
		go func() {
			defer wg.Done()

			sc := &DailyScheduleBusinessUnitSchedule{
				Name:     schedule.Name,
				City:     schedule.City,
				Address:  schedule.Address,
				Bookings: []*DailyScheduleBusinessUnitScheduleBooking{},
			}

			maestro.ReqAcquire()
			resp, err := ctx.Client.BookingClient.Search(
				unitID,
				schedule.ID,
				start.Format(time.RFC3339),
				end.Format(time.RFC3339),
				20,
				0,
			)
			maestro.ReqRelease()
			if err != nil {
				errMu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				errMu.Unlock()
				schedules[j] = sc
				return
			}

			books, _, err := ctx.Client.BookingClient.DecodeBookings(resp)
			if err != nil {
				errMu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				errMu.Unlock()
				schedules[j] = sc
				return
			}

			bookings := make([]*DailyScheduleBusinessUnitScheduleBooking, len(books))
			fillBookings(bookings, books)
			sc.Bookings = bookings

			schedules[j] = sc
		}()
	}

	wg.Wait()
	return firstErr
}

func fillBookings(bookings []*DailyScheduleBusinessUnitScheduleBooking, books []*model.Booking) {
	for i, booking := range books {
		bookings[i] = &DailyScheduleBusinessUnitScheduleBooking{
			Start:        booking.StartTime,
			End:          booking.EndTime,
			Label:        booking.ServiceLabel,
			Participants: make([]*DailyScheduleBusinessUnitScheduleBookingsParticipant, len(booking.Participants)),
		}
		fillParticipants(bookings[i], booking.Participants)
	}
}

func fillParticipants(b *DailyScheduleBusinessUnitScheduleBooking, participants map[string]string) {
	i := 0
	for name, phone := range participants {
		b.Participants[i] = &DailyScheduleBusinessUnitScheduleBookingsParticipant{
			Name:  name,
			Phone: phone,
		}
		i++
	}
}

func fetchAndApplyTimeFrameForView(ctx *maestro.MaestroContext) (time.Time, time.Time) {
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
