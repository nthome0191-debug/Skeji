package testutil

import (
	"skeji/pkg/model"
	"time"
)

type BusinessUnitBuilder struct {
	bu model.BusinessUnit
}

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

func (b *BusinessUnitBuilder) WithName(name string) *BusinessUnitBuilder {
	b.bu.Name = name
	return b
}

func (b *BusinessUnitBuilder) WithCities(cities ...string) *BusinessUnitBuilder {
	b.bu.Cities = cities
	return b
}

func (b *BusinessUnitBuilder) WithLabels(labels ...string) *BusinessUnitBuilder {
	b.bu.Labels = labels
	return b
}

func (b *BusinessUnitBuilder) WithAdminPhone(phone string) *BusinessUnitBuilder {
	b.bu.AdminPhone = phone
	return b
}

func (b *BusinessUnitBuilder) WithMaintainers(maintainers ...string) *BusinessUnitBuilder {
	b.bu.Maintainers = maintainers
	return b
}

func (b *BusinessUnitBuilder) WithPriority(priority int) *BusinessUnitBuilder {
	b.bu.Priority = priority
	return b
}

func (b *BusinessUnitBuilder) WithTimeZone(tz string) *BusinessUnitBuilder {
	b.bu.TimeZone = tz
	return b
}

func (b *BusinessUnitBuilder) WithWebsiteURL(url string) *BusinessUnitBuilder {
	b.bu.WebsiteURL = url
	return b
}

func (b *BusinessUnitBuilder) Build() model.BusinessUnit {
	return b.bu
}

func (b *BusinessUnitBuilder) BuildPtr() *model.BusinessUnit {
	bu := b.bu
	return &bu
}

func ValidBusinessUnit() model.BusinessUnit {
	return NewBusinessUnitBuilder().Build()
}

func MinimalBusinessUnit() model.BusinessUnit {
	return model.BusinessUnit{
		Name:       "Minimal Business",
		Cities:     []string{"Jerusalem"},
		Labels:     []string{"cafe"},
		AdminPhone: "+972541234567",
		TimeZone:   "Asia/Jerusalem",
	}
}

func EmptyBusinessUnit() model.BusinessUnit {
	return model.BusinessUnit{}
}

func InvalidPhoneBusinessUnit() model.BusinessUnit {
	return NewBusinessUnitBuilder().
		WithAdminPhone("invalid-phone").
		Build()
}

func UnsupportedCountryBusinessUnit() model.BusinessUnit {
	return NewBusinessUnitBuilder().
		WithAdminPhone("+441234567890").
		Build()
}

func MultiCityBusinessUnit() model.BusinessUnit {
	return NewBusinessUnitBuilder().
		WithCities("Tel Aviv", "Jerusalem", "Haifa").
		WithLabels("barbershop", "salon").
		Build()
}

func HighPriorityBusinessUnit() model.BusinessUnit {
	return NewBusinessUnitBuilder().
		WithName("Premium Business").
		WithPriority(100).
		Build()
}

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

func PartialBusinessUnitUpdate() model.BusinessUnitUpdate {
	return model.BusinessUnitUpdate{
		Name: "Partially Updated",
	}
}

func EmptyBusinessUnitUpdate() model.BusinessUnitUpdate {
	return model.BusinessUnitUpdate{}
}
