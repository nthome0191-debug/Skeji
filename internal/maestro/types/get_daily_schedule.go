package types

import (
	"fmt"
	"strings"
	"time"
)

// GetDailyScheduleInput defines the input parameters for getting daily schedule
type GetDailyScheduleInput struct {
	// Required fields
	Phone string `json:"phone" validate:"required,e164"`

	// Optional filter fields
	Cities []string `json:"cities,omitempty" validate:"omitempty,max=50"`
	Labels []string `json:"labels,omitempty" validate:"omitempty,max=10"`

	// Optional time range fields
	Start *time.Time `json:"start,omitempty"`
	End   *time.Time `json:"end,omitempty"`
}

// GetDailyScheduleOutput defines the output structure (for documentation)
type GetDailyScheduleOutput struct {
	Result *DailySchedule `json:"result"`
}

// DailySchedule represents the complete daily schedule structure
type DailySchedule struct {
	Units []*DailyScheduleBusinessUnit `json:"units"`
}

// DailyScheduleBusinessUnit represents a business unit with its schedules
type DailyScheduleBusinessUnit struct {
	Name      string                      `json:"name"`
	Labels    []string                    `json:"labels"`
	Schedules []*DailyScheduleSchedule    `json:"schedules"`
}

// DailyScheduleSchedule represents a schedule with its bookings
type DailyScheduleSchedule struct {
	Name     string                   `json:"name"`
	City     string                   `json:"city"`
	Address  string                   `json:"address"`
	Bookings []*DailyScheduleBooking  `json:"bookings"`
}

// DailyScheduleBooking represents a booking with participants
type DailyScheduleBooking struct {
	Start        time.Time                    `json:"start"`
	End          time.Time                    `json:"end"`
	Label        string                       `json:"label"`
	Participants []*DailyScheduleParticipant  `json:"participants"`
}

// DailyScheduleParticipant represents a booking participant
type DailyScheduleParticipant struct {
	Name  string `json:"name"`
	Phone string `json:"phone"`
}

// Validate checks if all required fields are present and valid
// Returns a detailed error message listing all validation failures
func (i *GetDailyScheduleInput) Validate() error {
	var errors []string

	// Validate required fields
	if strings.TrimSpace(i.Phone) == "" {
		errors = append(errors, "phone is required")
	}

	// Validate optional fields if provided
	if len(i.Cities) > 50 {
		errors = append(errors, "cities cannot have more than 50 items")
	}

	if len(i.Labels) > 10 {
		errors = append(errors, "labels cannot have more than 10 items")
	}

	// Validate time range if both are provided
	if i.Start != nil && i.End != nil {
		if i.End.Before(*i.Start) {
			errors = append(errors, "end time must be after start time")
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("validation failed:\n  - %s", strings.Join(errors, "\n  - "))
	}

	return nil
}

// ToMap converts the input struct to a map for use with MaestroContext
func (i *GetDailyScheduleInput) ToMap() map[string]any {
	m := map[string]any{
		"phone": i.Phone,
	}

	// Add optional filter fields
	if len(i.Cities) > 0 {
		m["cities"] = i.Cities
	}
	if len(i.Labels) > 0 {
		m["labels"] = i.Labels
	}

	// Add optional time fields
	if i.Start != nil {
		m["start"] = *i.Start
	}
	if i.End != nil {
		m["end"] = *i.End
	}

	return m
}

// FromMapGetDailySchedule creates a GetDailyScheduleInput from a map and validates it
func FromMapGetDailySchedule(input map[string]any) (*GetDailyScheduleInput, error) {
	i := &GetDailyScheduleInput{}

	// Extract required fields
	if phone, ok := input["phone"].(string); ok {
		i.Phone = phone
	}

	// Extract optional filter fields
	if cities, ok := input["cities"].([]string); ok {
		i.Cities = cities
	} else if citiesAny, ok := input["cities"].([]any); ok {
		i.Cities = make([]string, 0, len(citiesAny))
		for _, city := range citiesAny {
			if cityStr, ok := city.(string); ok {
				i.Cities = append(i.Cities, cityStr)
			}
		}
	}

	if labels, ok := input["labels"].([]string); ok {
		i.Labels = labels
	} else if labelsAny, ok := input["labels"].([]any); ok {
		i.Labels = make([]string, 0, len(labelsAny))
		for _, label := range labelsAny {
			if labelStr, ok := label.(string); ok {
				i.Labels = append(i.Labels, labelStr)
			}
		}
	}

	// Extract optional time fields
	if start, ok := input["start"].(time.Time); ok {
		i.Start = &start
	}

	if end, ok := input["end"].(time.Time); ok {
		i.End = &end
	}

	// Validate the parsed input
	if err := i.Validate(); err != nil {
		return nil, err
	}

	return i, nil
}
