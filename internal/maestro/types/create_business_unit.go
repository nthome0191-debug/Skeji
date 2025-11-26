package types

import (
	"fmt"
	"strings"
)

// CreateBusinessUnitInput defines the input parameters for creating a business unit with schedules
type CreateBusinessUnitInput struct {
	// Required Business Unit fields
	Name       string   `json:"name" validate:"required,min=2,max=100"`
	AdminPhone string   `json:"admin_phone" validate:"required,e164"`
	Cities     []string `json:"cities" validate:"required,min=1,max=50"`
	Labels     []string `json:"labels" validate:"required,min=1,max=10"`

	// Optional Business Unit fields
	TimeZone    string            `json:"time_zone,omitempty" validate:"omitempty,timezone"`
	WebsiteURLs []string          `json:"website_urls,omitempty" validate:"omitempty,max=5,dive,url"`
	Maintainers map[string]string `json:"maintainers,omitempty"`

	// Optional Schedule fields (applied to all city schedules)
	StartOfDay                *string  `json:"start_of_day,omitempty" validate:"omitempty,valid_time_range"`
	EndOfDay                  *string  `json:"end_of_day,omitempty" validate:"omitempty,valid_time_range"`
	WorkingDays               []string `json:"working_days,omitempty" validate:"omitempty,min=1,max=7"`
	ScheduleTimeZone          *string  `json:"schedule_time_zone,omitempty" validate:"omitempty,timezone"`
	DefaultMeetingDurationMin *int     `json:"default_meeting_duration_min,omitempty" validate:"omitempty,min=5,max=480"`
	DefaultBreakDurationMin   *int     `json:"default_break_duration_min,omitempty" validate:"omitempty,min=0,max=480"`
	MaxParticipantsPerSlot    *int     `json:"max_participants_per_slot,omitempty" validate:"omitempty,min=1,max=200"`

	SchedulePerCity bool `json:"schedule_per_city,omitempty" validate:"omitempty"`
}

// Validate checks if all required fields are present and valid
// Returns a detailed error message listing all validation failures
func (i *CreateBusinessUnitInput) Validate() error {
	var errors []string

	// Validate required fields
	if strings.TrimSpace(i.Name) == "" {
		errors = append(errors, "name is required")
	} else if len(i.Name) < 2 || len(i.Name) > 100 {
		errors = append(errors, "name must be between 2 and 100 characters")
	}

	if strings.TrimSpace(i.AdminPhone) == "" {
		errors = append(errors, "admin_phone is required")
	}

	if len(i.Cities) == 0 {
		errors = append(errors, "cities is required (at least one city)")
	} else if len(i.Cities) > 50 {
		errors = append(errors, "cities cannot have more than 50 items")
	}

	if len(i.Labels) == 0 {
		errors = append(errors, "labels is required (at least one label)")
	} else if len(i.Labels) > 10 {
		errors = append(errors, "labels cannot have more than 10 items")
	}

	// Validate optional fields if provided

	if len(i.WebsiteURLs) > 5 {
		errors = append(errors, "website_urls cannot have more than 5 items")
	}

	if i.DefaultMeetingDurationMin != nil {
		if *i.DefaultMeetingDurationMin < 5 {
			errors = append(errors, "default_meeting_duration_min must be at least 5")
		} else if *i.DefaultMeetingDurationMin > 480 {
			errors = append(errors, "default_meeting_duration_min cannot exceed 480")
		}
	}

	if i.DefaultBreakDurationMin != nil {
		if *i.DefaultBreakDurationMin < 0 {
			errors = append(errors, "default_break_duration_min must be non-negative")
		} else if *i.DefaultBreakDurationMin > 480 {
			errors = append(errors, "default_break_duration_min cannot exceed 480")
		}
	}

	if i.MaxParticipantsPerSlot != nil {
		if *i.MaxParticipantsPerSlot < 1 {
			errors = append(errors, "max_participants_per_slot must be at least 1")
		} else if *i.MaxParticipantsPerSlot > 200 {
			errors = append(errors, "max_participants_per_slot cannot exceed 200")
		}
	}

	if i.WorkingDays != nil {
		if len(i.WorkingDays) == 0 {
			errors = append(errors, "working_days must have at least 1 day when provided")
		} else if len(i.WorkingDays) > 7 {
			errors = append(errors, "working_days cannot have more than 7 days")
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("validation failed:\n  - %s", strings.Join(errors, "\n  - "))
	}

	return nil
}

// ToMap converts the input struct to a map for use with MaestroContext
func (i *CreateBusinessUnitInput) ToMap() map[string]any {
	m := map[string]any{
		"name":        i.Name,
		"admin_phone": i.AdminPhone,
		"cities":      i.Cities,
		"labels":      i.Labels,
	}

	// Add optional business unit fields
	if i.TimeZone != "" {
		m["time_zone"] = i.TimeZone
	}
	if len(i.WebsiteURLs) > 0 {
		m["website_urls"] = i.WebsiteURLs
	}
	if len(i.Maintainers) > 0 {
		m["maintainers"] = i.Maintainers
	}

	// Add optional schedule fields
	if i.StartOfDay != nil {
		m["start_of_day"] = *i.StartOfDay
	}
	if i.EndOfDay != nil {
		m["end_of_day"] = *i.EndOfDay
	}
	if len(i.WorkingDays) > 0 {
		m["working_days"] = i.WorkingDays
	}
	if i.ScheduleTimeZone != nil {
		m["schedule_time_zone"] = *i.ScheduleTimeZone
	}
	if i.DefaultMeetingDurationMin != nil {
		m["default_meeting_duration_min"] = *i.DefaultMeetingDurationMin
	}
	if i.DefaultBreakDurationMin != nil {
		m["default_break_duration_min"] = *i.DefaultBreakDurationMin
	}
	if i.MaxParticipantsPerSlot != nil {
		m["max_participants_per_slot"] = *i.MaxParticipantsPerSlot
	}

	// Add schedule behavior fields
	m["schedule_per_city"] = i.SchedulePerCity

	return m
}

// FromMap creates a CreateBusinessUnitInput from a map and validates it
func FromMapCreateBusinessUnit(input map[string]any) (*CreateBusinessUnitInput, error) {
	i := &CreateBusinessUnitInput{}

	// Extract required fields
	if name, ok := input["name"].(string); ok {
		i.Name = name
	}
	if adminPhone, ok := input["admin_phone"].(string); ok {
		i.AdminPhone = adminPhone
	}

	// Extract cities (handle both []string and []any from JSON)
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

	// Extract labels (handle both []string and []any from JSON)
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

	// Extract optional business unit fields
	if timeZone, ok := input["time_zone"].(string); ok {
		i.TimeZone = timeZone
	}

	if websiteURLsAny, ok := input["website_urls"].([]any); ok {
		i.WebsiteURLs = make([]string, 0, len(websiteURLsAny))
		for _, url := range websiteURLsAny {
			if urlStr, ok := url.(string); ok {
				i.WebsiteURLs = append(i.WebsiteURLs, urlStr)
			}
		}
	}

	if maintainers, ok := input["maintainers"].(map[string]string); ok {
		i.Maintainers = maintainers
	} else if maintainersAny, ok := input["maintainers"].(map[string]any); ok {
		i.Maintainers = make(map[string]string)
		for k, v := range maintainersAny {
			if vStr, ok := v.(string); ok {
				i.Maintainers[k] = vStr
			}
		}
	}

	// Extract optional schedule fields
	if startOfDay, ok := input["start_of_day"].(string); ok {
		i.StartOfDay = &startOfDay
	}
	if endOfDay, ok := input["end_of_day"].(string); ok {
		i.EndOfDay = &endOfDay
	}

	if workingDaysAny, ok := input["working_days"].([]any); ok {
		i.WorkingDays = make([]string, 0, len(workingDaysAny))
		for _, day := range workingDaysAny {
			if dayStr, ok := day.(string); ok {
				i.WorkingDays = append(i.WorkingDays, dayStr)
			}
		}
	}

	if scheduleTimeZone, ok := input["schedule_time_zone"].(string); ok {
		i.ScheduleTimeZone = &scheduleTimeZone
	}

	if val, ok := input["default_meeting_duration_min"]; ok && val != nil {
		switch v := val.(type) {
		case int:
			i.DefaultMeetingDurationMin = &v
		case int64:
			intVal := int(v)
			i.DefaultMeetingDurationMin = &intVal
		case float64:
			intVal := int(v)
			i.DefaultMeetingDurationMin = &intVal
		}
	}

	if val, ok := input["default_break_duration_min"]; ok && val != nil {
		switch v := val.(type) {
		case int:
			i.DefaultBreakDurationMin = &v
		case int64:
			intVal := int(v)
			i.DefaultBreakDurationMin = &intVal
		case float64:
			intVal := int(v)
			i.DefaultBreakDurationMin = &intVal
		}
	}

	if val, ok := input["max_participants_per_slot"]; ok && val != nil {
		switch v := val.(type) {
		case int:
			i.MaxParticipantsPerSlot = &v
		case int64:
			intVal := int(v)
			i.MaxParticipantsPerSlot = &intVal
		case float64:
			intVal := int(v)
			i.MaxParticipantsPerSlot = &intVal
		}
	}

	// Extract schedule behavior fields
	if schedulePerCity, ok := input["schedule_per_city"].(bool); ok {
		i.SchedulePerCity = schedulePerCity
	}

	// Validate the parsed input
	if err := i.Validate(); err != nil {
		return nil, err
	}

	return i, nil
}
