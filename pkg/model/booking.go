package model

import "time"

type Booking struct {
	ID           string    `json:"id,omitempty" bson:"_id,omitempty" validate:"omitempty,mongodb"`
	BusinessID   string    `json:"business_id" bson:"business_id" validate:"required,mongodb"`
	ScheduleID   string    `json:"schedule_id" bson:"schedule_id" validate:"required,mongodb"`
	ServiceLabel string    `json:"service_label" bson:"service_label" validate:"required,min=2,max=100"`
	StartTime    time.Time `json:"start_time" bson:"start_time" validate:"required,gt"`
	EndTime      time.Time `json:"end_time" bson:"end_time" validate:"required"`
	Capacity     int       `json:"capacity" bson:"capacity" validate:"required,min=1,max=200"`
	Participants []string  `json:"participants" bson:"participants" validate:"required,min=1,max=200,dive,required,e164"`
	Status       string    `json:"status" bson:"status" validate:"required,oneof=pending confirmed cancelled completed"`
	ManagedBy    string    `json:"managed_by" bson:"managed_by" validate:"required,e164"`
	CreatedAt    time.Time `json:"created_at" bson:"created_at" validate:"omitempty"`
}

type BookingUpdate struct {
	ServiceLabel string     `json:"service_label,omitempty" validate:"omitempty,min=2,max=100"`
	StartTime    *time.Time `json:"start_time,omitempty" validate:"omitempty,gt"`
	EndTime      *time.Time `json:"end_time,omitempty" validate:"omitempty"`
	Capacity     *int       `json:"capacity,omitempty" validate:"omitempty,min=1,max=200"`
	Participants *[]string  `json:"participants,omitempty" validate:"omitempty,min=1,max=200,dive,required,e164"`
	Status       string     `json:"status,omitempty" validate:"omitempty,oneof=pending confirmed cancelled completed"`
	ManagedBy    string     `json:"managed_by,omitempty" validate:"omitempty,e164"`
}
