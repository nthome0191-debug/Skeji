package flows

import (
	"fmt"
	maestro "skeji/internal/maestro/core"
	"skeji/pkg/config"
	"skeji/pkg/model"
	"skeji/pkg/sealer"
	"strings"
	"sync"
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
	businesses := []*Business{}
	var offset int64 = 0

	resp, err := ctx.Client.BusinessUnitClient.Search(cities, labels, MAX_RESULTS_PER_PAGE, offset)
	if err != nil {
		return err
	}
	units, metadata, err := ctx.Client.BusinessUnitClient.DecodeBusinessUnits(resp)
	if err != nil {
		return err
	}

	for len(businesses) < MAX_RESULTS_FOR_SEARCH && offset < metadata.TotalCount {
		var wg sync.WaitGroup
		var mu sync.Mutex

		for _, unit := range units {
			unit := unit

			// Early exit if we already have enough results
			mu.Lock()
			if len(businesses) >= MAX_RESULTS_FOR_SEARCH {
				mu.Unlock()
				break
			}
			mu.Unlock()

			wg.Add(1)
			maestro.RunWithRateLimitedConcurrency(func() {
				defer wg.Done()

				business := &Business{
					Name:     unit.Name,
					Phones:   []string{unit.AdminPhone},
					Branches: []*BusinessBranch{},
				}
				for phone := range unit.Maintainers {
					business.Phones = append(business.Phones, phone)
				}

				// Fetch branches for ALL cities in one batch call
				branches := fetchBranches(ctx, unit.ID, cities, start, end)
				if len(branches) > 0 {
					// Limit to max branches per unit
					if len(branches) > MAX_BRANCHES_PER_UNIT {
						branches = branches[:MAX_BRANCHES_PER_UNIT]
					}
					business.Branches = branches

					mu.Lock()
					if len(businesses) < MAX_RESULTS_FOR_SEARCH {
						businesses = append(businesses, business)
					}
					mu.Unlock()
				}
			})
		}

		wg.Wait()

		if len(businesses) >= MAX_RESULTS_FOR_SEARCH {
			break
		}

		offset += MAX_RESULTS_PER_PAGE
		resp, err = ctx.Client.BusinessUnitClient.Search(cities, labels, MAX_RESULTS_PER_PAGE, offset)
		if err != nil {
			ctx.Logger.Warn(fmt.Sprintf("business units search failed, err: %+v", err))
			continue
		}
		units, _, err = ctx.Client.BusinessUnitClient.DecodeBusinessUnits(resp)
		if err != nil {
			ctx.Logger.Warn(fmt.Sprintf("business units decode failed, err: %+v\nresp: %+v", err, resp))
			continue
		}
	}

	ctx.Output["result"] = businesses
	return nil
}

func fetchBranches(ctx *maestro.MaestroContext, buid string, cities []string, start, end time.Time) []*BusinessBranch {
	branches := []*BusinessBranch{}
	var offset int64 = 0

	// Use batch search to fetch all schedules across all cities in one call
	resp, err := ctx.Client.ScheduleClient.BatchSearch(buid, cities, MAX_RESULTS_PER_PAGE, offset)
	if err != nil {
		ctx.Logger.Warn(fmt.Sprintf("schedules batch search failed, err: %+v", err))
		return branches
	}
	schedules, metadata, err := ctx.Client.ScheduleClient.DecodeSchedules(resp)
	if err != nil {
		ctx.Logger.Warn(fmt.Sprintf("schedules decode failed, err: %+v\nresp: %+v", err, resp))
		return branches
	}

	for len(branches) < MAX_BRANCHES_PER_UNIT && offset < metadata.TotalCount {
		// Parallelize fetching open slots for all schedules
		type scheduleResult struct {
			branch *BusinessBranch
			hasSlots bool
		}

		results := make(chan scheduleResult, len(schedules))
		var wg sync.WaitGroup

		for _, schedule := range schedules {
			if len(branches) >= MAX_BRANCHES_PER_UNIT {
				break
			}

			schedule := schedule
			wg.Add(1)
			maestro.RunWithRateLimitedConcurrency(func() {
				defer wg.Done()

				branch := &BusinessBranch{
					City:        schedule.City,
					Address:     schedule.Address,
					WorkingDays: schedule.WorkingDays,
					StartOfDay:  schedule.StartOfDay,
					EndOfDay:    schedule.EndOfDay,
					OpenSlots:   []*OpenSlot{},
				}

				openSlots := fetchOpenSlots(ctx, buid, schedule, start, end)
				hasSlots := len(openSlots) > 0
				if hasSlots {
					for _, openSlot := range openSlots {
						if len(branch.OpenSlots) < MAX_OPEN_SLOTS_PER_BRANCH {
							branch.OpenSlots = append(branch.OpenSlots, openSlot)
						} else {
							break
						}
					}
				}

				results <- scheduleResult{branch: branch, hasSlots: hasSlots}
			})
		}

		go func() {
			wg.Wait()
			close(results)
		}()

		// Collect results
		for result := range results {
			if result.hasSlots && len(branches) < MAX_BRANCHES_PER_UNIT {
				branches = append(branches, result.branch)
			}
		}

		if len(branches) >= MAX_BRANCHES_PER_UNIT {
			break
		}

		offset += MAX_RESULTS_PER_PAGE
		resp, err = ctx.Client.ScheduleClient.BatchSearch(buid, cities, MAX_RESULTS_PER_PAGE, offset)
		if err != nil {
			ctx.Logger.Warn(fmt.Sprintf("schedules batch search failed, err: %+v", err))
			continue
		}
		schedules, metadata, err = ctx.Client.ScheduleClient.DecodeSchedules(resp)
		if err != nil {
			ctx.Logger.Warn(fmt.Sprintf("schedules decode failed, err: %+v\nresp: %+v", err, resp))
			continue
		}
	}
	return branches
}

func fetchOpenSlots(ctx *maestro.MaestroContext, buid string, sc *model.Schedule, start, end time.Time) []*OpenSlot {
	openSlots := []*OpenSlot{}
	var offset int64 = 0
	resp, err := ctx.Client.BookingClient.Search(buid, sc.ID, start.Format(time.RFC3339), end.Format(time.RFC3339), MAX_RESULTS_PER_PAGE, offset)
	if err != nil {
		ctx.Logger.Warn(fmt.Sprintf("booking search failed, err: %+v", err))
		return openSlots
	}
	bookings, metadata, err := ctx.Client.BookingClient.DecodeBookings(resp)
	if err != nil {
		ctx.Logger.Warn(fmt.Sprintf("booking decode failed, err: %+v\nresp: %+v", err, resp))
		return openSlots
	}
	batchId, err := sealer.CreateOpaqueToken(buid, sc.ID)
	if err != nil {
		ctx.Logger.Warn(fmt.Sprintf("create opaque token failed for [buid %v | scid %v] err: %v", buid, sc.ID, err))
		return openSlots
	}
	if len(bookings) == 0 {
		openSlots = append(openSlots, &OpenSlot{Start: start, End: end})
	} else {
		for len(openSlots) < MAX_OPEN_SLOTS_PER_BRANCH && offset < metadata.TotalCount {
			filteredSlots := filterSlots(bookings, start, end)
			if len(filteredSlots) > 0 {
				for _, slot := range filteredSlots {
					if len(openSlots) < MAX_OPEN_SLOTS_PER_BRANCH {
						openSlots = append(openSlots, slot)
					} else {
						break
					}
				}
			}
			if len(openSlots) >= MAX_OPEN_SLOTS_PER_BRANCH {
				break
			}
			offset += MAX_RESULTS_PER_PAGE
			resp, err = ctx.Client.BookingClient.Search(buid, sc.ID, start.Format(time.RFC3339), end.Format(time.RFC3339), MAX_RESULTS_PER_PAGE, offset)
			if err != nil {
				ctx.Logger.Warn(fmt.Sprintf("booking search failed, err: %+v", err))
				continue
			}
			bookings, metadata, err = ctx.Client.BookingClient.DecodeBookings(resp)
			if err != nil {
				ctx.Logger.Warn(fmt.Sprintf("booking decode failed, err: %+v\nresp: %+v", err, resp))
				continue
			}
		}
	}
	return normalizeSlots(ctx, batchId, openSlots, sc, start, end)
}

func filterSlots(bookings []*model.Booking, start, end time.Time) []*OpenSlot {
	openSlots := []*OpenSlot{}
	pStart := start
	for _, booking := range bookings {
		if booking.StartTime.After(pStart) {
			openSlots = append(openSlots, &OpenSlot{Start: pStart, End: booking.StartTime})
		}
		pStart = booking.EndTime
	}
	if end.After(pStart) {
		openSlots = append(openSlots, &OpenSlot{Start: pStart, End: end})
	}
	return openSlots
}

func normalizeSlots(ctx *maestro.MaestroContext, batchId string, slots []*OpenSlot, sc *model.Schedule, viewStart, viewEnd time.Time) []*OpenSlot {
	workWeek := buildWorkingDaysSet(sc.WorkingDays)
	openSlots := []*OpenSlot{}
	startToday, endToday, startTomorrow, endTomorrow, err := extractDailyFrames(sc.StartOfDay, sc.EndOfDay)
	if err != nil {
		ctx.Logger.Warn(fmt.Sprintf("extract daily frames failed: %v", err))
		return openSlots
	}
	for _, s := range slots {
		start1 := maxTime(s.Start, viewStart)
		end1 := minTime(s.End, viewEnd)
		if end1.Before(start1) {
			continue
		}
		part1Start := maxTime(start1, startToday)
		part1End := minTime(end1, endToday)

		if part1End.After(part1Start) {
			openSlot := &OpenSlot{
				ID:    batchId,
				Start: part1Start,
				End:   part1End,
			}
			if isLegitSlot(openSlot, sc, workWeek) {
				openSlots = append(openSlots, openSlot)
			}
		}

		if end1.After(startTomorrow) {
			part2Start := maxTime(start1, startTomorrow)
			part2End := minTime(end1, endTomorrow)

			if part2End.After(part2Start) {
				openSlot := &OpenSlot{
					ID:    batchId,
					Start: part2Start,
					End:   part2End,
				}
				if isLegitSlot(openSlot, sc, workWeek) {
					openSlots = append(openSlots, openSlot)
				}
			}
		}
	}
	return openSlots
}

func isLegitSlot(
	slot *OpenSlot,
	sc *model.Schedule,
	workWeek map[time.Weekday]bool,
) bool {
	if !workWeek[slot.Start.Weekday()] {
		return false
	}

	required := time.Duration(sc.DefaultMeetingDurationMin) * time.Minute
	return slot.End.Sub(slot.Start) >= required
}

func buildWorkingDaysSet(days []string) map[time.Weekday]bool {
	set := make(map[time.Weekday]bool)
	for _, d := range days {
		switch strings.ToLower(d) {
		case config.Sunday:
			set[time.Sunday] = true
		case config.Monday:
			set[time.Monday] = true
		case config.Tuesday:
			set[time.Tuesday] = true
		case config.Wednesday:
			set[time.Wednesday] = true
		case config.Thursday:
			set[time.Thursday] = true
		case config.Friday:
			set[time.Friday] = true
		case config.Saturday:
			set[time.Saturday] = true
		}
	}
	return set
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
	startParsed, err := time.Parse("15:04", startStr)
	if err != nil {
		return time.Time{}, time.Time{}, time.Time{}, time.Time{}, err
	}
	endParsed, err := time.Parse("15:04", endStr)
	if err != nil {
		return time.Time{}, time.Time{}, time.Time{}, time.Time{}, err
	}

	now := time.Now()
	year, month, day := now.Date()
	loc := now.Location()

	todayStart := time.Date(year, month, day, startParsed.Hour(), startParsed.Minute(), 0, 0, loc)
	todayEnd := time.Date(year, month, day, endParsed.Hour(), endParsed.Minute(), 0, 0, loc)

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
