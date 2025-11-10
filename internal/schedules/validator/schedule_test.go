package validator

import (
	"skeji/pkg/config"
	"skeji/pkg/logger"
	"skeji/pkg/model"
	"strings"
	"testing"
)

func TestValidateTimeRange(t *testing.T) {
	log := logger.New(logger.Config{
		Level:     "info",
		Format:    logger.JSON,
		AddSource: false,
		Service:   "test",
	})
	validator := NewScheduleValidator(log)

	tests := []struct {
		name        string
		startOfDay  string
		endOfDay    string
		wantError   bool
		description string
	}{
		{
			name:        "valid time range",
			startOfDay:  "09:00",
			endOfDay:    "18:00",
			wantError:   false,
			description: "standard business hours",
		},
		{
			name:        "edge case midnight to midnight",
			startOfDay:  "00:00",
			endOfDay:    "23:59",
			wantError:   false,
			description: "full day",
		},
		{
			name:        "invalid start hour",
			startOfDay:  "25:00",
			endOfDay:    "18:00",
			wantError:   true,
			description: "hour > 23",
		},
		{
			name:        "invalid end hour",
			startOfDay:  "09:00",
			endOfDay:    "25:00",
			wantError:   true,
			description: "hour > 23",
		},
		{
			name:        "invalid start minute",
			startOfDay:  "09:60",
			endOfDay:    "18:00",
			wantError:   true,
			description: "minute > 59",
		},
		{
			name:        "accepts format without leading zero",
			startOfDay:  "9:00",
			endOfDay:    "18:00",
			wantError:   false,
			description: "9:00 is valid time format",
		},
		{
			name:        "wrong format",
			startOfDay:  "09-00",
			endOfDay:    "18:00",
			wantError:   true,
			description: "dash instead of colon",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schedule := &model.Schedule{
				BusinessID:                "507f1f77bcf86cd799439011",
				Name:                      "Test Schedule",
				City:                      "Tel Aviv",
				Address:                   "123 Main St",
				StartOfDay:                tt.startOfDay,
				EndOfDay:                  tt.endOfDay,
				WorkingDays:               []config.Weekday{"Monday", "Tuesday"},
				DefaultMeetingDurationMin: 30,
				DefaultBreakDurationMin:   10,
				MaxParticipantsPerSlot:    5,
				TimeZone:                  "Asia/Jerusalem",
			}
			err := validator.Validate(schedule)
			if (err != nil) != tt.wantError {
				t.Errorf("%s: Validate() error = %v, wantError %v", tt.description, err, tt.wantError)
			}
		})
	}
}

func TestValidateWorkingDays(t *testing.T) {
	log := logger.New(logger.Config{
		Level:     "info",
		Format:    logger.JSON,
		AddSource: false,
		Service:   "test",
	})
	validator := NewScheduleValidator(log)

	tests := []struct {
		name        string
		workingDays []config.Weekday
		wantError   bool
		description string
	}{
		{
			name:        "valid weekdays",
			workingDays: []config.Weekday{"Sunday", "Monday", "Tuesday"},
			wantError:   false,
			description: "standard weekdays",
		},
		{
			name:        "all weekdays",
			workingDays: []config.Weekday{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"},
			wantError:   false,
			description: "7 days a week",
		},
		{
			name:        "invalid day",
			workingDays: []config.Weekday{"Sunday", "Funday"},
			wantError:   true,
			description: "invalid day name",
		},
		{
			name:        "empty working days",
			workingDays: []config.Weekday{},
			wantError:   true,
			description: "no working days",
		},
		{
			name:        "too many days",
			workingDays: []config.Weekday{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"},
			wantError:   true,
			description: "more than 7 days",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schedule := &model.Schedule{
				BusinessID:                "507f1f77bcf86cd799439011",
				Name:                      "Test Schedule",
				City:                      "Tel Aviv",
				Address:                   "123 Main St",
				StartOfDay:                "09:00",
				EndOfDay:                  "18:00",
				WorkingDays:               tt.workingDays,
				DefaultMeetingDurationMin: 30,
				DefaultBreakDurationMin:   10,
				MaxParticipantsPerSlot:    5,
				TimeZone:                  "Asia/Jerusalem",
			}
			err := validator.Validate(schedule)
			if (err != nil) != tt.wantError {
				t.Errorf("%s: Validate() error = %v, wantError %v", tt.description, err, tt.wantError)
			}
		})
	}
}

func TestValidateRequiredFields(t *testing.T) {
	log := logger.New(logger.Config{
		Level:     "info",
		Format:    logger.JSON,
		AddSource: false,
		Service:   "test",
	})
	validator := NewScheduleValidator(log)

	tests := []struct {
		name      string
		schedule  *model.Schedule
		wantError bool
		errorMsg  string
	}{
		{
			name: "missing business_id",
			schedule: &model.Schedule{
				Name:                      "Test Schedule",
				City:                      "Tel Aviv",
				Address:                   "123 Main St",
				StartOfDay:                "09:00",
				EndOfDay:                  "18:00",
				WorkingDays:               []config.Weekday{"Monday"},
				DefaultMeetingDurationMin: 30,
				DefaultBreakDurationMin:   10,
				MaxParticipantsPerSlot:    5,
				TimeZone:                  "Asia/Jerusalem",
			},
			wantError: true,
			errorMsg:  "business_id",
		},
		{
			name: "missing name",
			schedule: &model.Schedule{
				BusinessID:                "507f1f77bcf86cd799439011",
				City:                      "Tel Aviv",
				Address:                   "123 Main St",
				StartOfDay:                "09:00",
				EndOfDay:                  "18:00",
				WorkingDays:               []config.Weekday{"Monday"},
				DefaultMeetingDurationMin: 30,
				DefaultBreakDurationMin:   10,
				MaxParticipantsPerSlot:    5,
				TimeZone:                  "Asia/Jerusalem",
			},
			wantError: true,
			errorMsg:  "name",
		},
		{
			name: "missing city",
			schedule: &model.Schedule{
				BusinessID:                "507f1f77bcf86cd799439011",
				Name:                      "Test Schedule",
				Address:                   "123 Main St",
				StartOfDay:                "09:00",
				EndOfDay:                  "18:00",
				WorkingDays:               []config.Weekday{"Monday"},
				DefaultMeetingDurationMin: 30,
				DefaultBreakDurationMin:   10,
				MaxParticipantsPerSlot:    5,
				TimeZone:                  "Asia/Jerusalem",
			},
			wantError: true,
			errorMsg:  "city",
		},
		{
			name: "missing address",
			schedule: &model.Schedule{
				BusinessID:                "507f1f77bcf86cd799439011",
				Name:                      "Test Schedule",
				City:                      "Tel Aviv",
				StartOfDay:                "09:00",
				EndOfDay:                  "18:00",
				WorkingDays:               []config.Weekday{"Monday"},
				DefaultMeetingDurationMin: 30,
				DefaultBreakDurationMin:   10,
				MaxParticipantsPerSlot:    5,
				TimeZone:                  "Asia/Jerusalem",
			},
			wantError: true,
			errorMsg:  "address",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(tt.schedule)
			if (err != nil) != tt.wantError {
				t.Errorf("Validate() error = %v, wantError %v", err, tt.wantError)
			}
			if tt.wantError && err != nil {
				errStr := err.Error()
				// Field names in errors are capitalized (e.g., "BusinessID" not "business_id")
				expectedMsg := strings.ReplaceAll(tt.errorMsg, "_", "")
				if !strings.Contains(strings.ToLower(errStr), strings.ToLower(expectedMsg)) {
					t.Errorf("Expected error to contain %q (checking for %q), got %q", tt.errorMsg, expectedMsg, err.Error())
				}
			}
		})
	}
}

func TestValidateDurationBoundaries(t *testing.T) {
	log := logger.New(logger.Config{
		Level:     "info",
		Format:    logger.JSON,
		AddSource: false,
		Service:   "test",
	})
	validator := NewScheduleValidator(log)

	tests := []struct {
		name                      string
		defaultMeetingDurationMin int
		defaultBreakDurationMin   int
		maxParticipantsPerSlot    int
		wantError                 bool
	}{
		{
			name:                      "valid durations",
			defaultMeetingDurationMin: 30,
			defaultBreakDurationMin:   10,
			maxParticipantsPerSlot:    5,
			wantError:                 false,
		},
		{
			name:                      "valid minimum values",
			defaultMeetingDurationMin: 30,
			defaultBreakDurationMin:   10,
			maxParticipantsPerSlot:    1,
			wantError:                 false,
		},
		{
			name:                      "maximum meeting duration",
			defaultMeetingDurationMin: 480,
			defaultBreakDurationMin:   480,
			maxParticipantsPerSlot:    200,
			wantError:                 false,
		},
		{
			name:                      "meeting duration too small",
			defaultMeetingDurationMin: 4,
			defaultBreakDurationMin:   10,
			maxParticipantsPerSlot:    5,
			wantError:                 true,
		},
		{
			name:                      "meeting duration too large",
			defaultMeetingDurationMin: 481,
			defaultBreakDurationMin:   10,
			maxParticipantsPerSlot:    5,
			wantError:                 true,
		},
		{
			name:                      "break duration negative",
			defaultMeetingDurationMin: 30,
			defaultBreakDurationMin:   -1,
			maxParticipantsPerSlot:    5,
			wantError:                 true,
		},
		{
			name:                      "max participants zero",
			defaultMeetingDurationMin: 30,
			defaultBreakDurationMin:   10,
			maxParticipantsPerSlot:    0,
			wantError:                 true,
		},
		{
			name:                      "max participants too large",
			defaultMeetingDurationMin: 30,
			defaultBreakDurationMin:   10,
			maxParticipantsPerSlot:    201,
			wantError:                 true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schedule := &model.Schedule{
				BusinessID:                "507f1f77bcf86cd799439011",
				Name:                      "Test Schedule",
				City:                      "Tel Aviv",
				Address:                   "123 Main St",
				StartOfDay:                "09:00",
				EndOfDay:                  "18:00",
				WorkingDays:               []config.Weekday{"Monday"},
				DefaultMeetingDurationMin: tt.defaultMeetingDurationMin,
				DefaultBreakDurationMin:   tt.defaultBreakDurationMin,
				MaxParticipantsPerSlot:    tt.maxParticipantsPerSlot,
				TimeZone:                  "Asia/Jerusalem",
			}
			err := validator.Validate(schedule)
			if (err != nil) != tt.wantError {
				t.Errorf("Validate() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}
