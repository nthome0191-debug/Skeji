package validator

import (
	"errors"
	"fmt"
	"skeji/pkg/logger"
	"skeji/pkg/model"
	"strings"

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
	return fmt.Sprintf("validation failed: %d error(s)", len(v))
}

type BusinessUnitValidator struct {
	validate *validator.Validate
	logger   *logger.Logger
}

func NewBusinessUnitValidator(log *logger.Logger) *BusinessUnitValidator {
	v := validator.New()

	if err := v.RegisterValidation("supported_country", validateSupportedCountry); err != nil {
		log.Fatal("Failed to register 'supported_country' validator",
			"error", err,
		)
	}

	log.Info("Business unit validator initialized successfully")

	return &BusinessUnitValidator{
		validate: v,
		logger:   log,
	}
}

func validateSupportedCountry(fl validator.FieldLevel) bool {
	phone := strings.TrimSpace(fl.Field().String())

	supportedPrefixes := []string{
		"+972", "972", // Israel
		"+1", // United States and Canada
	}

	for _, prefix := range supportedPrefixes {
		if strings.HasPrefix(phone, prefix) {
			return true
		}
	}

	return false
}

func (v *BusinessUnitValidator) Validate(bu *model.BusinessUnit) error {
	if err := v.validate.Struct(bu); err != nil {
		var validationErrs validator.ValidationErrors
		if errors.As(err, &validationErrs) {
			return v.translateValidationErrors(validationErrs)
		}
		return err
	}
	return nil
}

func (v *BusinessUnitValidator) translateValidationErrors(errs validator.ValidationErrors) ValidationErrors {
	var validationErrors ValidationErrors

	for _, err := range errs {
		message := err.Error()

		switch err.Tag() {
		case "supported_country":
			message = "phone number must be from a supported country, stated country is not supported by app yet"
		case "timezone":
			message = fmt.Sprintf("invalid timezone '%s', must be a valid IANA timezone (e.g., America/New_York, Asia/Jerusalem, UTC)", err.Value())
		case "e164":
			message = "phone number must be in E.164 format (e.g., +972501234567)"
		case "url":
			message = fmt.Sprintf("invalid URL format '%s'", err.Value())
		case "startswith":
			message = fmt.Sprintf("URL must start with '%s'", err.Param())
		case "required":
			message = fmt.Sprintf("%s is required", err.Field())
		case "min":
			message = fmt.Sprintf("%s must be at least %s", err.Field(), err.Param())
		case "max":
			message = fmt.Sprintf("%s must be at most %s", err.Field(), err.Param())
		}

		validationErrors = append(validationErrors, ValidationError{
			Field:   err.Field(),
			Message: message,
		})
	}

	return validationErrors
}
