package validator

import (
	"errors"
	"fmt"
	"skeji/pkg/logger"
	"skeji/pkg/model"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
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
	if err := v.RegisterValidation("valid_working_days", validateWorkingDays); err != nil {
		log.Fatal("Failed to register 'valid_working_days' validator", "error", err)
	}

	log.Info("Schedule validator initialized successfully")

	return &ScheduleValidator{
		validate: v,
		logger:   log,
	}
}

func validateTimeRange(fl validator.FieldLevel) bool {

	dayFrame := strings.TrimSpace(fl.Field().String())

	var err error

	if dayFrame != "" {
		_, err = time.Parse("15:04", dayFrame)
		if err != nil {
			return false
		}
		var startHour, startMin int
		if _, err := fmt.Sscanf(dayFrame, "%02d:%02d", &startHour, &startMin); err != nil {
			return false
		}
		if startHour < 0 || startHour > 23 {
			return false
		}
		if startMin < 0 || startMin > 59 {
			return false
		}
	}

	return true
}

func validateWorkingDays(fl validator.FieldLevel) bool {
	days, ok := fl.Field().Interface().([]string)
	if !ok || len(days) == 0 {
		return false
	}

	validDays := map[string]struct{}{
		"sunday": {}, "monday": {}, "tuesday": {}, "wednesday": {},
		"thursday": {}, "friday": {}, "saturday": {},
	}

	for _, d := range days {
		if _, valid := validDays[strings.ToLower(strings.TrimSpace(d))]; !valid {
			return false
		}
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
	return nil
}

func (v *ScheduleValidator) translateValidationErrors(errs validator.ValidationErrors) ValidationErrors {
	var validationErrors ValidationErrors

	for _, err := range errs {
		message := err.Error()

		switch err.Tag() {
		case "required":
			message = fmt.Sprintf("%s is required", err.Field())
		case "min":
			message = fmt.Sprintf("%s must be at least %s characters", err.Field(), err.Param())
		case "max":
			message = fmt.Sprintf("%s must be at most %s characters", err.Field(), err.Param())
		case "valid_time_range":
			message = "end_of_day must be after start_of_day and both must be in HH:MM 24-hour format"
		case "valid_working_days":
			message = "working_days must contain only valid weekday names (Sunday-Saturday)"
		}

		validationErrors = append(validationErrors, ValidationError{
			Field:   err.Field(),
			Message: message,
		})
	}

	return validationErrors
}
