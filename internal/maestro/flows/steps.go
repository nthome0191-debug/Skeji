package flows

import (
	maestro "skeji/internal/maestro/core"
)

const (
	BUSINESS_UNITS = "business_units"
)

func ListPhoneRelatedBusinessUnits(ctx *maestro.MaestroContext) error {
	// phone := ctx.Input["phone"].(string)
	// resp, err := ctx.Client.BusinessUnitClient.GetByPhone(phone, config.DefaultMaxBusinessUnitsPerAdminPhone, 0)
	// if err != nil {
	// 	return err
	// }
	// todo: convert client response to units
	// ctx.Output[BUSINESS_UNITS] = units
	return nil
}

func ListBusinessUnitsRelatedSchedules(ctx *maestro.MaestroContext) error {
	// units := ctx.Output[BUSINESS_UNITS].([]*model.BusinessUnit)
	// schedules := map[string]map[string][]*model.Schedule{}
	// for _, unit := range units {
	// 	for _, city := range unit.Cities {
	// 		resp, err := ctx.Client.ScheduleClient.Search(unit.ID, city, config.DefaultMaxSchedulesPerBusinessUnits, 0)
	// 		if err != nil {
	// 			return err
	// 		}
	// 		// convert resp to schedules
	// 	}
	// }
	return nil
}
