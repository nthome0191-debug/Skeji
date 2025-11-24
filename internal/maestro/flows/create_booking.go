package flows

import (
	maestro "skeji/internal/maestro/core"
	"skeji/pkg/sealer"
	"time"
)

// Requester phone, requester name, start_time, slot_id
func CreateBooking(ctx *maestro.MaestroContext) error {
	requesterPhone := ctx.ExtractString("requester_phone")
	requesterName := ctx.ExtractString("requester_name")
	slotId := ctx.ExtractString("slot_id")
	startTime, err := ctx.ExtractTime("start_time")
	if err != nil {
		return err
	}
	useUserEndTime := false
	var endTime time.Time
	buid, schid, err := sealer.ParseOpaqueToken(slotId)
	if err != nil {
		return err
	}
	resp, err := ctx.Client.BusinessUnitClient.GetByID(buid)
	if err != nil {
		return err
	}
	bu, err := ctx.Client.BusinessUnitClient.DecodeBusinessUnit(resp)
	if err != nil {
		return err
	}
	if _, isMaintainer := bu.Maintainers[requesterPhone]; isMaintainer || bu.AdminPhone == requesterPhone {
		if _, endTimeProvided := ctx.Input["end_time"]; endTimeProvided {
			endTime, err = ctx.ExtractTime("end_time")
			if err == nil {
				useUserEndTime = true
			}
		}
	}
	return nil
}
