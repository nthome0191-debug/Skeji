package validator

import (
	"errors"
	"fmt"
	"skeji/pkg/model"

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
}

func NewBusinessUnitValidator() *BusinessUnitValidator {
	v := validator.New()

	// TODO: Register custom validators here
	// Example: v.RegisterValidation("custom_tag", customValidationFunc)

	return &BusinessUnitValidator{
		validate: v,
	}
}

func (v *BusinessUnitValidator) Validate(bu *model.BusinessUnit) error {
	if err := v.validate.Struct(bu); err != nil {
		var validationErrs validator.ValidationErrors
		if errors.As(err, &validationErrs) {
			return v.translateValidationErrors(validationErrs)
		}
		return err
	}

	if err := v.validateBusinessRules(bu); err != nil {
		return err
	}

	return nil
}

func (v *BusinessUnitValidator) translateValidationErrors(errs validator.ValidationErrors) ValidationErrors {
	var validationErrors ValidationErrors

	for _, err := range errs {
		// TODO: Implement field name translation and user-friendly messages
		validationErrors = append(validationErrors, ValidationError{
			Field:   err.Field(),
			Message: err.Error(), // Placeholder - should be human-readable
		})
	}

	return validationErrors
}

func (v *BusinessUnitValidator) validateBusinessRules(bu *model.BusinessUnit) error {
	// TODO: Implement custom business rules here
	// Examples:
	// - Check if cities are valid/supported
	// - Validate phone number format beyond e164
	// - Check for duplicate cities/labels
	// - Validate timezone against IANA database
	// - Business-specific rules (e.g., certain labels require certain cities)

	return nil
}
