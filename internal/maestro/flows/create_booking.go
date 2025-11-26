package flows

import (
	"fmt"
	"net/http"
	maestro "skeji/internal/maestro/core"
	"skeji/pkg/config"
	"skeji/pkg/model"
	"skeji/pkg/sealer"
	"time"
)

func CreateBooking(ctx *maestro.MaestroContext) error {
	requesterPhone := ctx.ExtractString("requester_phone")
	if maestro.IsMissing(requesterPhone) {
		return maestro.MissingParamErr("requester_phone")
	}
	slotId := ctx.ExtractString("slot_id")
	if maestro.IsMissing(slotId) {
		return maestro.MissingParamErr("slot_id")
	}
	startTime, err := ctx.ExtractTime("start_time")
	if err != nil {
		return err
	}
	isMaintainerRequest := false
	endTimeProvided := false
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
		isMaintainerRequest = true
		if _, endTimeProvided = ctx.Input["end_time"]; endTimeProvided {
			endTime, err = ctx.ExtractTime("end_time")
			if err != nil {
				endTimeProvided = false
			}
		}
	}
	requesterName := ctx.ExtractString("requester_name")
	if !isMaintainerRequest {
		if maestro.IsMissing(requesterName) {
			return maestro.MissingParamErr("requester_name")
		}
	} else {
		if len(requesterName) == 0 {
			requesterName = "admin"
		}
	}
	booking := &model.Booking{
		BusinessID:   buid,
		ScheduleID:   schid,
		StartTime:    startTime,
		Participants: map[string]string{requesterName: requesterPhone},
		ManagedBy:    map[string]string{},
	}
	if isMaintainerRequest {
		booking.Status = config.Confirmed
		booking.ManagedBy[requesterName] = requesterPhone
	} else {
		booking.Status = config.Pending
	}
	resp, err = ctx.Client.ScheduleClient.GetByID(schid)
	if err != nil {
		return err
	}
	schedule, err := ctx.Client.ScheduleClient.DecodeSchedule(resp)
	if err != nil {
		return err
	}
	booking.Capacity = schedule.MaxParticipantsPerSlot
	if endTimeProvided {
		booking.EndTime = endTime
	} else {
		booking.EndTime = booking.StartTime.Add(time.Duration(schedule.DefaultMeetingDurationMin) * time.Minute)
	}
	resp, err = ctx.Client.BookingClient.Create(booking)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("%+v", resp.ToString())
	}
	createdBooking, err := ctx.Client.BookingClient.DecodeBooking(resp)
	if err != nil {
		return err
	}
	ctx.Output["booking"] = createdBooking
	return nil
}
