package model

import "time"

// BookingLock represents an advisory lock for preventing concurrent booking creation
// This is a lightweight mechanism to prevent race conditions during booking overlap checks
type BookingLock struct {
	ID        string    `bson:"_id" json:"id"`
	ExpiresAt time.Time `bson:"expires_at" json:"expires_at"`
	CreatedAt time.Time `bson:"created_at" json:"created_at"`
}
