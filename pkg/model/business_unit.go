package model

import "time"

type BusinessUnit struct {
	ID          string    `bson:"_id,omitempty" validate:"omitempty,mongodb"`
	Name        string    `bson:"name" validate:"required,min=2,max=100"`
	Cities      []string  `bson:"cities" validate:"required,min=1,max=50,dive,required"`
	Labels      []string  `bson:"labels" validate:"required,min=1,max=10,dive,required"`
	AdminPhone  string    `bson:"admin_phone" validate:"required,e164"`
	Maintainers []string  `bson:"maintainers" validate:"omitempty,dive,required"`
	Priority    int       `bson:"priority" validate:"omitempty,min=0"`
	TimeZone    string    `bson:"time_zone" validate:"omitempty,timezone"`
	CreatedAt   time.Time `bson:"created_at" validate:"omitempty"`
}
