package model

import (
	"time"
)

type Booking struct {
	ID           string            `json:"id,omitempty" bson:"_id,omitempty" validate:"omitempty,mongodb"`
	BusinessID   string            `json:"business_id" bson:"business_id" validate:"required,mongodb"`
	ScheduleID   string            `json:"schedule_id" bson:"schedule_id" validate:"required,mongodb"`
	ServiceLabel string            `json:"service_label" bson:"service_label" validate:"required,min=2,max=100"`
	StartTime    time.Time         `json:"start_time" bson:"start_time" validate:"required"`
	EndTime      time.Time         `json:"end_time" bson:"end_time" validate:"required,gtfield=StartTime"`
	Capacity     int               `json:"capacity" bson:"capacity" validate:"required,min=1,max=200"`
	Participants map[string]string `json:"participants" bson:"participants" validate:"omitempty,participants_map"`
	Status       string            `json:"status" bson:"status" validate:"required,oneof=pending confirmed cancelled"`
	ManagedBy    map[string]string `json:"managed_by" bson:"managed_by" validate:"required,participants_map"`
	CreatedAt    time.Time         `json:"created_at" bson:"created_at" validate:"omitempty"`
}

type BookingUpdate struct {
	ServiceLabel string             `json:"service_label,omitempty" validate:"omitempty,min=2,max=100"`
	StartTime    *time.Time         `json:"start_time,omitempty" validate:"omitempty"`
	EndTime      *time.Time         `json:"end_time,omitempty" validate:"omitempty,gtfield=StartTime"`
	Capacity     *int               `json:"capacity,omitempty" validate:"omitempty,min=1,max=200"`
	Participants *map[string]string `json:"participants,omitempty" validate:"omitempty,participants_map"`
	Status       string             `json:"status,omitempty" validate:"omitempty,oneof=pending confirmed cancelled"`
	ManagedBy    map[string]string  `json:"managed_by,omitempty" validate:"omitempty,participants_map"`
}
