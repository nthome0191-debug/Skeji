package flows

import (
	maestro "skeji/internal/maestro/core"
	"skeji/pkg/model"
	"time"
)

const (
	MAX_RESULTS_FOR_SEARCH    = 5
	MAX_BRANCHES_PER_UNIT     = 3
	MAX_OPEN_SLOTS_PER_BRANCH = 3

	MAX_RESULTS_PER_PAGE = 200
)

type OpenSlot struct {
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
	Phone    string
	Branches []*BusinessBranch
}

// require: city, labels
// optional: start_time, end+time, address
func SearchBusiness(ctx *maestro.MaestroContext) error {
	cities := ctx.ExtractStringList("cities")
	labels := ctx.ExtractStringList("labels")
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
		for _, unit := range units {
			business := &Business{
				Name:     unit.Name,
				Phone:    unit.AdminPhone,
				Branches: []*BusinessBranch{},
			}
			addBusiness := false
			for _, city := range cities {
				branches := fetchBranches(ctx, unit.ID, city, start, end)
				if len(branches) > 0 {
					addBusiness = true
					for _, branch := range branches {
						if len(business.Branches) < MAX_BRANCHES_PER_UNIT {
							business.Branches = append(business.Branches, branch)
						} else {
							break
						}
					}
				}
				if len(business.Branches) >= MAX_BRANCHES_PER_UNIT {
					break
				}
			}
			if addBusiness {
				businesses = append(businesses, business)
			}
			if len(businesses) >= MAX_RESULTS_FOR_SEARCH {
				break
			}
		}
		if len(businesses) >= MAX_RESULTS_FOR_SEARCH {
			break
		}
		offset += MAX_RESULTS_PER_PAGE
		resp, err = ctx.Client.BusinessUnitClient.Search(cities, labels, MAX_RESULTS_PER_PAGE, offset)
		if err != nil {
			continue
		}
		units, _, err = ctx.Client.BusinessUnitClient.DecodeBusinessUnits(resp)
		if err != nil {
			continue
		}
	}
	ctx.Output["result"] = businesses
	return nil
}

func fetchBranches(ctx *maestro.MaestroContext, buid string, city string, start, end time.Time) []*BusinessBranch {
	branches := []*BusinessBranch{}
	var offset int64 = 0
	resp, err := ctx.Client.ScheduleClient.Search(buid, city, MAX_RESULTS_PER_PAGE, offset)
	if err != nil {
		return branches
	}
	schedules, metadata, err := ctx.Client.ScheduleClient.DecodeSchedules(resp)
	if err != nil {
		return branches
	}
	for len(branches) < MAX_BRANCHES_PER_UNIT && offset < metadata.TotalCount {
		for _, schedule := range schedules {
			branch := &BusinessBranch{
				City:        schedule.City,
				Address:     schedule.Address,
				WorkingDays: schedule.WorkingDays,
				StartOfDay:  schedule.StartOfDay,
				EndOfDay:    schedule.EndOfDay,
				OpenSlots:   []*OpenSlot{},
			}
			addBranch := false
			openSlots := fetchOpenSlots(ctx, buid, schedule.ID, start, end)
			if len(openSlots) > 0 {
				addBranch = true
				for _, openSlot := range openSlots {
					if len(branch.OpenSlots) < MAX_OPEN_SLOTS_PER_BRANCH {
						branch.OpenSlots = append(branch.OpenSlots, openSlot)
					} else {
						break
					}
				}
			}
			if addBranch {
				branches = append(branches, branch)
			}
			if len(branches) >= MAX_BRANCHES_PER_UNIT {
				break
			}
		}
		if len(branches) >= MAX_BRANCHES_PER_UNIT {
			break
		}
		offset += MAX_RESULTS_PER_PAGE
		resp, err = ctx.Client.ScheduleClient.Search(buid, city, MAX_RESULTS_PER_PAGE, offset)
		if err != nil {
			continue
		}
		schedules, metadata, err = ctx.Client.ScheduleClient.DecodeSchedules(resp)
		if err != nil {
			continue
		}
	}
	return branches
}

func fetchOpenSlots(ctx *maestro.MaestroContext, buid string, scid string, start, end time.Time) []*OpenSlot {
	openSlots := []*OpenSlot{}
	var offset int64 = 0
	resp, err := ctx.Client.BookingClient.Search(buid, scid, start.Format(time.RFC3339), end.Format(time.RFC3339), MAX_RESULTS_PER_PAGE, offset)
	if err != nil {
		return openSlots
	}
	bookings, metadata, err := ctx.Client.BookingClient.DecodeBookings(resp)
	if err != nil {
		return openSlots
	}
	for len(openSlots) < MAX_OPEN_SLOTS_PER_BRANCH && offset < metadata.TotalCount {
		if len(bookings) == 0 {
			openSlots = append(openSlots, &OpenSlot{Start: start, End: end})
			break
		}
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
		resp, err = ctx.Client.BookingClient.Search(buid, scid, start.Format(time.RFC3339), end.Format(time.RFC3339), MAX_RESULTS_PER_PAGE, offset)
		if err != nil {
			continue
		}
		bookings, metadata, err = ctx.Client.BookingClient.DecodeBookings(resp)
		if err != nil {
			continue
		}
	}
	return normalizeSlots(openSlots)
}

func filterSlots(bookings []*model.Booking, start, end time.Time) []*OpenSlot {
	openSlots := []*OpenSlot{}
	return openSlots
}

func normalizeSlots(openSlots []*OpenSlot) []*OpenSlot {
	return openSlots
}

func fetchAndApplyTimeFrameForSearch(ctx *maestro.MaestroContext) (time.Time, time.Time) {
	now := time.Now()

	start, err := ctx.ExtractTime("start")
	if err != nil {
		start = now
	}

	maxStart := now.Add(48 * time.Hour)
	minStart := now

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
