package flows

import (
	maestro "skeji/internal/maestro/core"
	"time"
)

const (
	MAX_RESULTS_FOR_SEARCH    = 5
	MAX_BRANCHES_PER_UNIT     = 3
	MAX_OPEN_SLOTS_PER_BRANCH = 3
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
	openSlots   []*OpenSlot
}

type Business struct {
	Name     string
	Phone    string
	Branches []*BusinessBranch
}

// require: city, labels
// optional: start_time, address
func SearchBusiness(ctx *maestro.MaestroContext) error {
	// cities := ctx.ExtractStringList("cities")
	// labels := ctx.ExtractStringList("labels")
	// start, end := fetchAndApplyTimeFrameForSearch(ctx)

	// businesses := make([]*Business, MAX_RESULTS_FOR_SEARCH)

	// offset := 0
	// maxPerPage := 200
	// resp, err := ctx.Client.BusinessUnitClient.Search(cities, labels, maxPerPage, int64(offset))
	// if err != nil {
	// 	return err
	// }

	// i := 0
	// units, metadata, err := ctx.Client.BusinessUnitClient.DecodeBusinessUnits(resp)
	// if err != nil {
	// 	return err
	// }
	// for _, unit := range units {
	// 	if i >= MAX_RESULTS_FOR_SEARCH {
	// 		break
	// 	}
	// 	for _, city := range cities {
	// 		resp, err := ctx.Client.ScheduleClient.Search(unit.ID, city, maxPerPage, 0)
	// 		if err != nil {
	// 			return err
	// 		}
	// 		schedules, metadata, err := ctx.Client.ScheduleClient.DecodeSchedules(resp)
	// 		if err != nil {
	// 			return err
	// 		}
	// 		for _, schedule := range schedules {
	// 			resp, err := ctx.Client.BookingClient.Search(unit.ID, schedule.ID, start.Format(time.RFC3339), end.Format(time.RFC3339), maxPerPage, 0)
	// 			if err != nil {
	// 				return err
	// 			}
	// 		}
	// 	}
	// }

	return nil
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
