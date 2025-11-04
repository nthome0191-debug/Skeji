package testutil

import (
	"skeji/pkg/model"
	"time"
)

// BusinessUnitBuilder provides a fluent API for creating test business units
type BusinessUnitBuilder struct {
	bu model.BusinessUnit
}

// NewBusinessUnitBuilder creates a new builder with sensible defaults
func NewBusinessUnitBuilder() *BusinessUnitBuilder {
	return &BusinessUnitBuilder{
		bu: model.BusinessUnit{
			Name:        "Test Business",
			Cities:      []string{"Tel Aviv"},
			Labels:      []string{"barbershop"},
			AdminPhone:  "+972501234567",
			Maintainers: []string{},
			Priority:    10,
			TimeZone:    "Asia/Jerusalem",
			WebsiteURL:  "https://example.com",
			CreatedAt:   time.Now(),
		},
	}
}

// WithName sets the business name
func (b *BusinessUnitBuilder) WithName(name string) *BusinessUnitBuilder {
	b.bu.Name = name
	return b
}

// WithCities sets the cities
func (b *BusinessUnitBuilder) WithCities(cities ...string) *BusinessUnitBuilder {
	b.bu.Cities = cities
	return b
}

// WithLabels sets the labels
func (b *BusinessUnitBuilder) WithLabels(labels ...string) *BusinessUnitBuilder {
	b.bu.Labels = labels
	return b
}

// WithAdminPhone sets the admin phone
func (b *BusinessUnitBuilder) WithAdminPhone(phone string) *BusinessUnitBuilder {
	b.bu.AdminPhone = phone
	return b
}

// WithMaintainers sets the maintainers
func (b *BusinessUnitBuilder) WithMaintainers(maintainers ...string) *BusinessUnitBuilder {
	b.bu.Maintainers = maintainers
	return b
}

// WithPriority sets the priority
func (b *BusinessUnitBuilder) WithPriority(priority int) *BusinessUnitBuilder {
	b.bu.Priority = priority
	return b
}

// WithTimeZone sets the timezone
func (b *BusinessUnitBuilder) WithTimeZone(tz string) *BusinessUnitBuilder {
	b.bu.TimeZone = tz
	return b
}

// WithWebsiteURL sets the website URL
func (b *BusinessUnitBuilder) WithWebsiteURL(url string) *BusinessUnitBuilder {
	b.bu.WebsiteURL = url
	return b
}

// Build returns the constructed business unit
func (b *BusinessUnitBuilder) Build() model.BusinessUnit {
	return b.bu
}

// BuildPtr returns a pointer to the constructed business unit
func (b *BusinessUnitBuilder) BuildPtr() *model.BusinessUnit {
	bu := b.bu
	return &bu
}

// Predefined test fixtures

// ValidBusinessUnit returns a valid business unit for testing
func ValidBusinessUnit() model.BusinessUnit {
	return NewBusinessUnitBuilder().Build()
}

// MinimalBusinessUnit returns a business unit with only required fields
func MinimalBusinessUnit() model.BusinessUnit {
	return model.BusinessUnit{
		Name:       "Minimal Business",
		Cities:     []string{"Jerusalem"},
		Labels:     []string{"cafe"},
		AdminPhone: "+972541234567",
		TimeZone:   "Asia/Jerusalem",
	}
}

// EmptyBusinessUnit returns a business unit with empty/zero values
func EmptyBusinessUnit() model.BusinessUnit {
	return model.BusinessUnit{}
}

// InvalidPhoneBusinessUnit returns a business unit with invalid phone
func InvalidPhoneBusinessUnit() model.BusinessUnit {
	return NewBusinessUnitBuilder().
		WithAdminPhone("invalid-phone").
		Build()
}

// UnsupportedCountryBusinessUnit returns a business unit with unsupported country phone
func UnsupportedCountryBusinessUnit() model.BusinessUnit {
	return NewBusinessUnitBuilder().
		WithAdminPhone("+441234567890"). // UK phone (not supported)
		Build()
}

// MultiCityBusinessUnit returns a business unit serving multiple cities
func MultiCityBusinessUnit() model.BusinessUnit {
	return NewBusinessUnitBuilder().
		WithCities("Tel Aviv", "Jerusalem", "Haifa").
		WithLabels("barbershop", "salon").
		Build()
}

// HighPriorityBusinessUnit returns a high priority business unit
func HighPriorityBusinessUnit() model.BusinessUnit {
	return NewBusinessUnitBuilder().
		WithName("Premium Business").
		WithPriority(100).
		Build()
}

// BusinessUnitUpdate helpers

// ValidBusinessUnitUpdate returns a valid update payload
func ValidBusinessUnitUpdate() model.BusinessUnitUpdate {
	priority := 20
	websiteURL := "https://updated.com"
	maintainers := []string{"+972501111111"}

	return model.BusinessUnitUpdate{
		Name:        "Updated Business",
		Cities:      []string{"Haifa"},
		Labels:      []string{"updated"},
		AdminPhone:  "+972509999999",
		Priority:    &priority,
		TimeZone:    "Asia/Jerusalem",
		WebsiteURL:  &websiteURL,
		Maintainers: &maintainers,
	}
}

// PartialBusinessUnitUpdate returns an update with only some fields
func PartialBusinessUnitUpdate() model.BusinessUnitUpdate {
	return model.BusinessUnitUpdate{
		Name: "Partially Updated",
	}
}

// EmptyBusinessUnitUpdate returns an update with all empty values
func EmptyBusinessUnitUpdate() model.BusinessUnitUpdate {
	return model.BusinessUnitUpdate{}
}
