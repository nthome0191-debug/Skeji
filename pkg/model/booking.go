package model

import "time"

type Booking struct {
	ID           string    `bson:"_id,omitempty"`
	BusinessID   string    `bson:"business_id"`
	ScheduleID   string    `bson:"schedule_id"`
	ServiceLabel string    `bson:"service_label"`
	StartTime    time.Time `bson:"start_time"`
	EndTime      time.Time `bson:"end_time"`
	Capacity     int       `bson:"capacity"`
	Participants []string  `bson:"participants"`
	Status       string    `bson:"status"`
	ManagedBy    string    `bson:"managed_by"`
	CreatedAt    time.Time `bson:"created_at"`
}
