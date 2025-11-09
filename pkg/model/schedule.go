package model

import "time"

type Schedule struct {
	ID                     string    `bson:"_id,omitempty"`
	BusinessID             string    `bson:"business_id"`
	Name                   string    `bson:"name"`
	City                   string    `bson:"city"`
	Address                string    `bson:"address"`
	StartOfDay             string    `bson:"start_of_day"`
	EndOfDay               string    `bson:"end_of_day"`
	WorkingDays            []string  `bson:"working_days"`
	DefaultMeetingDuration int       `bson:"default_meeting_duration_min"`
	DefaultBreakDuration   int       `bson:"default_break_duration_min"`
	MaxParticipantsPerSlot int       `bson:"max_participants_per_slot"`
	Exceptions             []string  `bson:"exceptions"`
	CreatedAt              time.Time `bson:"created_at"`
}

func (s *Schedule) GetID() string {
	return s.ID
}
