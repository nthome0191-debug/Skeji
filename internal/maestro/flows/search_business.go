package flows

import (
	"fmt"
	maestro "skeji/internal/maestro/core"
	"skeji/pkg/config"
	"skeji/pkg/model"
	"skeji/pkg/sealer"
	"strings"
	"time"
)

const (
	MAX_RESULTS_FOR_SEARCH    = 5
	MAX_BRANCHES_PER_UNIT     = 3
	MAX_OPEN_SLOTS_PER_BRANCH = 3

	MAX_RESULTS_PER_PAGE = 200
)

type OpenSlot struct {
	ID    string
	Start time.Time
	End   time.Time
}

type BusinessBranch struct {
	City        string
	Address     string
	WorkingDays []string
	StartOfDay  string
	EndOfDay    string
	OpenSlots   []*OpenSlot
}

type Business struct {
	Name     string
	Phones   []string
	Branches []*BusinessBranch
}

func SearchBusiness(ctx *maestro.MaestroContext) error {
	cities := ctx.ExtractStringList("cities")
	labels := ctx.ExtractStringList("labels")
	if len(cities) == 0 || len(labels) == 0 {
		return fmt.Errorf("at least one label and one city must be specified")
	}
	start, end := fetchAndApplyTimeFrameForSearch(ctx)
	businesses := searchBusinessUnits(ctx, cities, labels, start, end)
	ctx.Output["result"] = businesses
	return nil
}

func searchBusinessUnits(ctx *maestro.MaestroContext, cities, labels []string, start, end time.Time) []*Business {
	result := []*Business{}
	var offset int64 = 0

	resp, err := ctx.Client.BusinessUnitClient.Search(cities, labels, MAX_RESULTS_PER_PAGE, offset)
	if err != nil {
		return result
	}

	units, metadata, err := ctx.Client.BusinessUnitClient.DecodeBusinessUnits(resp)
	if err != nil {
		return result
	}

	for len(result) < MAX_RESULTS_FOR_SEARCH && offset < metadata.TotalCount {
		for _, unit := range units {
			business := buildBusinessEntry(unit)
			if collectBranchesForBusiness(ctx, business, unit.ID, cities, start, end) {
				result = append(result, business)
			}
			if len(result) >= MAX_RESULTS_FOR_SEARCH {
				return result
			}
		}

		offset += MAX_RESULTS_PER_PAGE
		resp, err = ctx.Client.BusinessUnitClient.Search(cities, labels, MAX_RESULTS_PER_PAGE, offset)
		if err != nil {
			ctx.Logger.Warn(fmt.Sprintf("business units search failed, err: %+v", err))
			continue
		}
		units, _, err = ctx.Client.BusinessUnitClient.DecodeBusinessUnits(resp)
		if err != nil {
			ctx.Logger.Warn(fmt.Sprintf("business units decode failed, err: %+v", err))
			continue
		}
	}

	return result
}

func buildBusinessEntry(unit *model.BusinessUnit) *Business {
	b := &Business{
		Name:     unit.Name,
		Phones:   []string{unit.AdminPhone},
		Branches: []*BusinessBranch{},
	}
	for phone := range unit.Maintainers {
		b.Phones = append(b.Phones, phone)
	}
	return b
}

func collectBranchesForBusiness(ctx *maestro.MaestroContext, b *Business, buid string, cities []string, start, end time.Time) bool {
	added := false
	for _, city := range cities {
		branches := fetchBranches(ctx, buid, city, start, end)
		for _, branch := range branches {
			if len(b.Branches) < MAX_BRANCHES_PER_UNIT {
				b.Branches = append(b.Branches, branch)
				added = true
			}
		}
		if len(b.Branches) >= MAX_BRANCHES_PER_UNIT {
			break
		}
	}
	return added
}

func fetchBranches(ctx *maestro.MaestroContext, buid string, city string, start, end time.Time) []*BusinessBranch {
	result := []*BusinessBranch{}
	var offset int64 = 0

	resp, err := ctx.Client.ScheduleClient.Search(buid, city, MAX_RESULTS_PER_PAGE, offset)
	if err != nil {
		return result
	}

	schedules, metadata, err := ctx.Client.ScheduleClient.DecodeSchedules(resp)
	if err != nil {
		return result
	}

	for len(result) < MAX_BRANCHES_PER_UNIT && offset < metadata.TotalCount {
		for _, sc := range schedules {
			branch := buildBranchEntry(sc)
			openSlots := fetchOpenSlots(ctx, buid, sc, start, end)
			for _, s := range openSlots {
				if len(branch.OpenSlots) < MAX_OPEN_SLOTS_PER_BRANCH {
					branch.OpenSlots = append(branch.OpenSlots, s)
				}
			}
			if len(branch.OpenSlots) > 0 {
				result = append(result, branch)
			}
			if len(result) >= MAX_BRANCHES_PER_UNIT {
				return result
			}
		}

		offset += MAX_RESULTS_PER_PAGE
		resp, err = ctx.Client.ScheduleClient.Search(buid, city, MAX_RESULTS_PER_PAGE, offset)
		if err != nil {
			continue
		}
		schedules, _, err = ctx.Client.ScheduleClient.DecodeSchedules(resp)
		if err != nil {
			continue
		}
	}

	return result
}

func buildBranchEntry(sc *model.Schedule) *BusinessBranch {
	return &BusinessBranch{
		City:        sc.City,
		Address:     sc.Address,
		WorkingDays: sc.WorkingDays,
		StartOfDay:  sc.StartOfDay,
		EndOfDay:    sc.EndOfDay,
		OpenSlots:   []*OpenSlot{},
	}
}

func fetchOpenSlots(ctx *maestro.MaestroContext, buid string, sc *model.Schedule, start, end time.Time) []*OpenSlot {
	result := []*OpenSlot{}
	var offset int64 = 0

	resp, err := ctx.Client.BookingClient.Search(buid, sc.ID, start.Format(time.RFC3339), end.Format(time.RFC3339), MAX_RESULTS_PER_PAGE, offset)
	if err != nil {
		return result
	}

	bookings, metadata, err := ctx.Client.BookingClient.DecodeBookings(resp)
	if err != nil {
		return result
	}

	batchId, err := sealer.CreateOpaqueToken(buid, sc.ID)
	if err != nil {
		return result
	}

	raw := computeRawSlots(bookings, start, end)

	for len(result) < MAX_OPEN_SLOTS_PER_BRANCH && offset < metadata.TotalCount {
		normalized := normalizeSlots(ctx, batchId, raw, sc, start, end)
		for _, s := range normalized {
			if len(result) < MAX_OPEN_SLOTS_PER_BRANCH {
				result = append(result, s)
			}
		}
		if len(result) >= MAX_OPEN_SLOTS_PER_BRANCH {
			return result
		}

		offset += MAX_RESULTS_PER_PAGE
		resp, err = ctx.Client.BookingClient.Search(buid, sc.ID, start.Format(time.RFC3339), end.Format(time.RFC3339), MAX_RESULTS_PER_PAGE, offset)
		if err != nil {
			continue
		}
		bookings, metadata, err = ctx.Client.BookingClient.DecodeBookings(resp)
		if err != nil {
			continue
		}
		raw = computeRawSlots(bookings, start, end)
	}

	return result
}

func computeRawSlots(bookings []*model.Booking, start, end time.Time) []*OpenSlot {
	slots := []*OpenSlot{}
	if len(bookings) == 0 {
		return []*OpenSlot{{Start: start, End: end}}
	}
	p := start
	for _, b := range bookings {
		if b.StartTime.After(p) {
			slots = append(slots, &OpenSlot{Start: p, End: b.StartTime})
		}
		p = b.EndTime
	}
	if end.After(p) {
		slots = append(slots, &OpenSlot{Start: p, End: end})
	}
	return slots
}

func normalizeSlots(ctx *maestro.MaestroContext, batchId string, slots []*OpenSlot, sc *model.Schedule, viewStart, viewEnd time.Time) []*OpenSlot {
	workWeek := buildWorkingDaysSet(sc.WorkingDays)
	result := []*OpenSlot{}

	todayStart, todayEnd, tomorrowStart, tomorrowEnd, err := extractDailyFrames(sc.StartOfDay, sc.EndOfDay)
	if err != nil {
		return result
	}

	for _, s := range slots {
		clippedStart := maxTime(s.Start, viewStart)
		clippedEnd := minTime(s.End, viewEnd)
		if clippedEnd.Before(clippedStart) {
			continue
		}

		slot1Start := maxTime(clippedStart, todayStart)
		slot1End := minTime(clippedEnd, todayEnd)
		if slot1End.After(slot1Start) {
			slot := &OpenSlot{ID: batchId, Start: slot1Start, End: slot1End}
			if isLegitSlot(slot, sc, workWeek) {
				result = append(result, slot)
			}
		}

		if clippedEnd.After(tomorrowStart) {
			slot2Start := maxTime(clippedStart, tomorrowStart)
			slot2End := minTime(clippedEnd, tomorrowEnd)
			if slot2End.After(slot2Start) {
				slot := &OpenSlot{ID: batchId, Start: slot2Start, End: slot2End}
				if isLegitSlot(slot, sc, workWeek) {
					result = append(result, slot)
				}
			}
		}
	}

	return result
}

func isLegitSlot(slot *OpenSlot, sc *model.Schedule, workWeek map[time.Weekday]bool) bool {
	if !workWeek[slot.Start.Weekday()] {
		return false
	}
	required := time.Duration(sc.DefaultMeetingDurationMin) * time.Minute
	return slot.End.Sub(slot.Start) >= required
}

func buildWorkingDaysSet(days []string) map[time.Weekday]bool {
	m := make(map[time.Weekday]bool)
	for _, d := range days {
		switch strings.ToLower(d) {
		case config.Sunday:
			m[time.Sunday] = true
		case config.Monday:
			m[time.Monday] = true
		case config.Tuesday:
			m[time.Tuesday] = true
		case config.Wednesday:
			m[time.Wednesday] = true
		case config.Thursday:
			m[time.Thursday] = true
		case config.Friday:
			m[time.Friday] = true
		case config.Saturday:
			m[time.Saturday] = true
		}
	}
	return m
}

func minTime(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}

func maxTime(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}

func extractDailyFrames(startStr, endStr string) (time.Time, time.Time, time.Time, time.Time, error) {
	s, err := time.Parse("15:04", startStr)
	if err != nil {
		return time.Time{}, time.Time{}, time.Time{}, time.Time{}, err
	}
	e, err := time.Parse("15:04", endStr)
	if err != nil {
		return time.Time{}, time.Time{}, time.Time{}, time.Time{}, err
	}
	now := time.Now()
	y, m, d := now.Date()
	loc := now.Location()

	todayStart := time.Date(y, m, d, s.Hour(), s.Minute(), 0, 0, loc)
	todayEnd := time.Date(y, m, d, e.Hour(), e.Minute(), 0, 0, loc)
	if todayEnd.Before(todayStart) {
		todayEnd = todayStart.Add(24 * time.Hour)
	}
	tomorrowStart := todayStart.Add(24 * time.Hour)
	tomorrowEnd := todayEnd.Add(24 * time.Hour)

	return todayStart, todayEnd, tomorrowStart, tomorrowEnd, nil
}

func fetchAndApplyTimeFrameForSearch(ctx *maestro.MaestroContext) (time.Time, time.Time) {
	now := time.Now()

	start, err := ctx.ExtractTime("start")
	if err != nil {
		start = now
	}
	maxStart := now.Add(10 * time.Hour)
	minStart := now
	if start.After(maxStart) {
		start = maxStart
	}
	if start.Before(minStart) {
		start = minStart
	}

	end, err := ctx.ExtractTime("end")
	if err != nil {
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
