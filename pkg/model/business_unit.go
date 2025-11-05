package model

import "time"

type BusinessUnit struct {
	ID          string    `bson:"_id,omitempty" validate:"omitempty,mongodb"`
	Name        string    `bson:"name" validate:"required,min=2,max=100"`
	Cities      []string  `bson:"cities" validate:"required,min=1,max=50,dive,required"`
	Labels      []string  `bson:"labels" validate:"required,min=1,max=10,dive,required"`
	AdminPhone  string    `bson:"admin_phone" validate:"required,e164,supported_country"`
	Maintainers []string  `bson:"maintainers" validate:"omitempty,dive,required"`
	Priority    int64     `bson:"priority" validate:"omitempty,min=0"`
	TimeZone    string    `bson:"time_zone" validate:"omitempty,timezone"`
	WebsiteURL  string    `json:"website_url" bson:"website_url,omitempty" validate:"omitempty,url,startswith=https://"`
	CreatedAt   time.Time `bson:"created_at" validate:"omitempty"`
}

type BusinessUnitUpdate struct {
	Name        string    `json:"name,omitempty" validate:"omitempty,min=2,max=100"`
	Cities      []string  `json:"cities,omitempty" validate:"omitempty,min=1,max=50,dive,required"`
	Labels      []string  `json:"labels,omitempty" validate:"omitempty,min=1,max=10,dive,required"`
	AdminPhone  string    `json:"admin_phone,omitempty" validate:"omitempty,e164,supported_country"`
	Maintainers *[]string `json:"maintainers,omitempty" validate:"omitempty,dive,required"`
	Priority    *int64    `json:"priority,omitempty" validate:"omitempty,min=0"`
	TimeZone    string    `json:"time_zone,omitempty" validate:"omitempty,timezone"`
	WebsiteURL  *string   `json:"website_url,omitempty" validate:"omitempty,url,startswith=https://"`
}
