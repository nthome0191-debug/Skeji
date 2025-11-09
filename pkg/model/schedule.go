package model

import "time"

type Schedule struct {
	ID                     string    `json:"id,omitempty" bson:"_id,omitempty" validate:"omitempty,mongodb"`
	BusinessID             string    `json:"business_id" bson:"business_id" validate:"required,mongodb"`
	Name                   string    `json:"name" bson:"name" validate:"required,min=2,max=100"`
	City                   string    `json:"city" bson:"city" validate:"required,min=2,max=50"`
	Address                string    `json:"address" bson:"address" validate:"required,min=2,max=200"`
	StartOfDay             string    `json:"start_of_day" bson:"start_of_day" validate:"required,time"`
	EndOfDay               string    `json:"end_of_day" bson:"end_of_day" validate:"required,time,gtfield=StartOfDay"`
	WorkingDays            []string  `json:"working_days" bson:"working_days" validate:"required,min=1,max=7,dive,oneof=Sunday Monday Tuesday Wednesday Thursday Friday Saturday"`
	DefaultMeetingDuration int       `json:"default_meeting_duration_min" bson:"default_meeting_duration_min" validate:"required,min=5,max=480"`
	DefaultBreakDuration   int       `json:"default_break_duration_min" bson:"default_break_duration_min" validate:"required,min=0,max=480"`
	MaxParticipantsPerSlot int       `json:"max_participants_per_slot" bson:"max_participants_per_slot" validate:"required,min=1,max=200"`
	Exceptions             []string  `json:"exceptions,omitempty" bson:"exceptions" validate:"omitempty"`
	CreatedAt              time.Time `json:"created_at" bson:"created_at" validate:"omitempty"`
}

type ScheduleUpdate struct {
	Name                   string    `json:"name,omitempty" validate:"omitempty,min=2,max=100"`
	City                   string    `json:"city,omitempty" validate:"omitempty,min=2,max=50"`
	Address                string    `json:"address,omitempty" validate:"omitempty,min=2,max=200"`
	StartOfDay             string    `json:"start_of_day,omitempty" validate:"omitempty,time"`
	EndOfDay               string    `json:"end_of_day,omitempty" validate:"omitempty,time,gtfield=StartOfDay"`
	WorkingDays            []string  `json:"working_days,omitempty" validate:"omitempty,min=1,max=7,dive,oneof=Sunday Monday Tuesday Wednesday Thursday Friday Saturday"`
	DefaultMeetingDuration *int      `json:"default_meeting_duration_min,omitempty" validate:"omitempty,min=5,max=480"`
	DefaultBreakDuration   *int      `json:"default_break_duration_min,omitempty" validate:"omitempty,min=0,max=480"`
	MaxParticipantsPerSlot *int      `json:"max_participants_per_slot,omitempty" validate:"omitempty,min=1,max=200"`
	Exceptions             *[]string `json:"exceptions,omitempty" validate:"omitempty"`
}
