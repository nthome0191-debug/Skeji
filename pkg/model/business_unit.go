package model

import "time"

type BusinessUnit struct {
	ID          string    `json:"id,omitempty" bson:"_id,omitempty" validate:"omitempty,mongodb"`
	Name        string    `json:"name" bson:"name" validate:"required,min=2,max=100"`
	Cities      []string  `json:"cities" bson:"cities" validate:"required,min=1,max=50,dive,required"`
	Labels      []string  `json:"labels" bson:"labels" validate:"required,min=1,max=10,dive,required"`
	AdminPhone  string    `json:"admin_phone" bson:"admin_phone" validate:"required,e164,supported_country"`
	Maintainers []string  `json:"maintainers,omitempty" bson:"maintainers" validate:"omitempty,dive,required"`
	Priority    int64     `json:"priority,omitempty" bson:"priority" validate:"omitempty,min=0"`
	TimeZone    string    `json:"time_zone,omitempty" bson:"time_zone" validate:"omitempty,timezone"`
	WebsiteURL  string    `json:"website_url,omitempty" bson:"website_url,omitempty" validate:"omitempty,url,startswith=https://"`
	CreatedAt   time.Time `json:"created_at,omitempty" bson:"created_at" validate:"omitempty"`
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
