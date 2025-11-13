package validator

import (
	"errors"
	"fmt"
	"regexp"
	"skeji/pkg/logger"
	"skeji/pkg/model"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
)

var (
	phoneRegex = regexp.MustCompile(`^(?:|\+[1-9]\d{7,14})$`)
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

type BookingValidator struct {
	validate *validator.Validate
	logger   *logger.Logger
}

func NewBookingValidator(log *logger.Logger) *BookingValidator {
	v := validator.New()

	if err := v.RegisterValidation("participants_map", validateParticipantsMap); err != nil {
		log.Fatal("Failed to register 'participants_map' validator",
			"error", err,
		)
	}

	log.Info("Booking validator initialized successfully")

	return &BookingValidator{
		validate: v,
		logger:   log,
	}
}

func validateParticipantsMap(fl validator.FieldLevel) bool {
	value := fl.Field()
	fmt.Println(fl.FieldName())

	if value.IsNil() {
		return false
	}

	participants, ok := value.Interface().(map[string]string)
	if !ok {
		return false
	}

	n := len(participants)
	if n == 0 {
		return true
	}
	if n < 1 || n > 200 {
		return false
	}

	for name, phone := range participants {
		if phone == "" || name == "" || !phoneRegex.MatchString(phone) {
			return false
		}
	}
	return true
}

func (v *BookingValidator) Validate(booking *model.Booking) error {
	if err := v.validate.Struct(booking); err != nil {
		var validationErrs validator.ValidationErrors
		if errors.As(err, &validationErrs) {
			return v.translateValidationErrors(validationErrs)
		}
		return err
	}

	if !booking.EndTime.After(booking.StartTime) {
		return ValidationErrors{
			ValidationError{
				Field:   "EndTime",
				Message: "end_time must be after start_time",
			},
		}
	}

	if len(booking.Participants) > booking.Capacity {
		return ValidationErrors{
			ValidationError{
				Field:   "Participants",
				Message: fmt.Sprintf("participants count (%d) exceeds capacity (%d)", len(booking.Participants), booking.Capacity),
			},
		}
	}

	if booking.StartTime.Before(time.Now()) {
		return ValidationErrors{
			ValidationError{
				Field:   "StratTime",
				Message: "start_time cannot be in the past",
			},
		}
	}

	return nil
}

func (v *BookingValidator) ValidateUpdate(update *model.BookingUpdate) error {
	if err := v.validate.Struct(update); err != nil {
		var validationErrs validator.ValidationErrors
		if errors.As(err, &validationErrs) {
			return v.translateValidationErrors(validationErrs)
		}
		return err
	}

	if update.StartTime != nil && update.EndTime != nil {
		if !update.EndTime.After(*update.StartTime) {
			return ValidationErrors{
				ValidationError{
					Field:   "EndTime",
					Message: "end_time must be after start_time",
				},
			}
		}
	}

	return nil
}

func (v *BookingValidator) translateValidationErrors(errs validator.ValidationErrors) ValidationErrors {
	var validationErrors ValidationErrors

	for _, err := range errs {
		message := err.Error()

		switch err.Tag() {
		case "required":
			message = fmt.Sprintf("%s is required", err.Field())
		case "min":
			message = fmt.Sprintf("%s must be at least %s", err.Field(), err.Param())
		case "max":
			message = fmt.Sprintf("%s must be at most %s", err.Field(), err.Param())
		case "mongodb":
			message = fmt.Sprintf("%s must be a valid MongoDB ObjectID", err.Field())
		case "e164":
			message = fmt.Sprintf("%s must be in E.164 format (e.g., +972501234567)", err.Field())
		case "oneof":
			message = fmt.Sprintf("%s must be one of: %s", err.Field(), err.Param())
		case "gt":
			message = fmt.Sprintf("%s must be greater than current time", err.Field())
		}

		validationErrors = append(validationErrors, ValidationError{
			Field:   err.Field(),
			Message: message,
		})
	}

	return validationErrors
}
