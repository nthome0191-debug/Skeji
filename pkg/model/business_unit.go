package model

import "time"

type BusinessUnit struct {
	ID          string    `bson:"_id,omitempty"`
	Name        string    `bson:"name"`
	Cities      []string  `bson:"cities"`
	Labels      []string  `bson:"labels"`
	AdminPhone  string    `bson:"admin_phone"`
	Maintainers []string  `bson:"maintainers"`
	Priority    int       `bson:"priority"`
	TimeZone    string    `bson:"time_zone"`
	CreatedAt   time.Time `bson:"created_at"`
}
