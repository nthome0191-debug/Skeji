package validator

import (
	"errors"
	"fmt"
	"skeji/pkg/config"
	"skeji/pkg/logger"
	"skeji/pkg/model"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
)

var (
	validDays = map[string]struct{}{
		config.Sunday: {}, config.Monday: {}, config.Tuesday: {},
		config.Wednesday: {}, config.Thursday: {}, config.Friday: {}, config.Saturday: {},
	}
)

type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (v ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", v.Field, v.Message)
}

type ValidationErrors []ValidationError

func (v ValidationErrors) Error() string {
	if len(v) == 0 {
		return ""
	}
	var messages []string
	for _, err := range v {
		messages = append(messages, err.Error())
	}
	return fmt.Sprintf("validation failed: %d error(s): [%s]", len(v), strings.Join(messages, "; "))
}

type ScheduleValidator struct {
	validate *validator.Validate
	logger   *logger.Logger
}

func NewScheduleValidator(log *logger.Logger) *ScheduleValidator {
	v := validator.New()
	if err := v.RegisterValidation("valid_time_range", validateTimeRange); err != nil {
		log.Fatal("Failed to register 'valid_time_range' validator", "error", err)
	}
	if err := v.RegisterValidation("valid_week_days", validWeekDays); err != nil {
		log.Fatal("Failed to register 'valid_week_days' validator", "error", err)
	}
	log.Info("Schedule validator initialized successfully")
	return &ScheduleValidator{validate: v, logger: log}
}

func validateTimeRange(fl validator.FieldLevel) bool {
	val := strings.TrimSpace(fl.Field().String())
	if val == "" {
		return true
	}
	t, err := time.Parse("15:04", val)
	if err != nil {
		return false
	}
	h, m := t.Hour(), t.Minute()
	return h >= 0 && h <= 23 && m >= 0 && m <= 59
}

func validWeekDays(fl validator.FieldLevel) bool {
	day, ok := fl.Field().Interface().(string)
	if !ok {
		return false
	}
	day = strings.ToLower(strings.TrimSpace(day))
	if _, exists := validDays[day]; !exists {
		return false
	}

	return true
}

func (v *ScheduleValidator) Validate(sc *model.Schedule) error {
	if err := v.validate.Struct(sc); err != nil {
		var validationErrs validator.ValidationErrors
		if errors.As(err, &validationErrs) {
			return v.translateValidationErrors(validationErrs)
		}
		return err
	}

	startOfDay, err := time.Parse("15:04", sc.StartOfDay)
	if err != nil {
		return err
	}
	endOfDay, err := time.Parse("15:04", sc.EndOfDay)
	if err != nil {
		return err
	}
	if !endOfDay.After(startOfDay) {
		return ValidationErrors{{
			Field:   "end_of_day",
			Message: "end_of_day must be after start_of_day",
		}}
	}
	if len(sc.WorkingDays) == 0 || len(sc.WorkingDays) > 7 {
		return ValidationErrors{{
			Field:   "working_days",
			Message: "working_days lenght must be between 1 to 7",
		}}
	}
	if len(sc.Exceptions) > 10 {
		return ValidationErrors{{
			Field:   "exceptions",
			Message: "excpetions lenght must be no more than 10 distinct items",
		}}
	}
	return nil
}

func (v *ScheduleValidator) translateValidationErrors(errs validator.ValidationErrors) ValidationErrors {
	var out ValidationErrors

	fieldMap := map[string]string{
		"Name": "name", "City": "city", "Address": "address",
		"StartOfDay": "start_of_day", "EndOfDay": "end_of_day",
		"WorkingDays":               "working_days",
		"DefaultMeetingDurationMin": "default_meeting_duration_min",
		"DefaultBreakDurationMin":   "default_break_duration_min",
		"MaxParticipantsPerSlot":    "max_participants_per_slot",
	}

	for _, err := range errs {
		message := ""
		switch err.Tag() {
		case "required":
			message = fmt.Sprintf("%s is required", err.Field())
		case "min":
			switch err.Field() {
			case "Name", "City", "Address":
				message = fmt.Sprintf("%s must be at least %s chars", err.Field(), err.Param())
			case "StartOfDay", "EndOfDay", "DefaultMeetingDurationMin", "DefaultBreakDurationMin":
				message = fmt.Sprintf("%s must be at least %s minutes", err.Field(), err.Param())
			case "WorkingDays":
				message = fmt.Sprintf("%s must be at least %s weekdays", err.Field(), err.Param())
			case "MaxParticipantsPerSlot":
				message = fmt.Sprintf("%s must be at least %s participants", err.Field(), err.Param())
			}
		case "max":
			switch err.Field() {
			case "Name", "City", "Address":
				message = fmt.Sprintf("%s must be at most %s chars", err.Field(), err.Param())
			case "StartOfDay", "EndOfDay", "DefaultMeetingDurationMin", "DefaultBreakDurationMin":
				message = fmt.Sprintf("%s must be at most %s minutes", err.Field(), err.Param())
			case "WorkingDays":
				message = fmt.Sprintf("%s must be at most %s weekdays", err.Field(), err.Param())
			case "MaxParticipantsPerSlot":
				message = fmt.Sprintf("%s must be at most %s participants", err.Field(), err.Param())
			}
		case "valid_time_range":
			message = fmt.Sprintf("%s must be in valid 24-hour HH:MM format", err.Field())
		default:
			message = err.Error()
		}

		jsonField := fieldMap[err.Field()]
		if jsonField == "" {
			jsonField = strings.ToLower(err.Field())
		}

		out = append(out, ValidationError{Field: jsonField, Message: message})
	}

	return out
}
