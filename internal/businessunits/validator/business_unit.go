package validator

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"regexp"
	"skeji/pkg/config"
	"skeji/pkg/locale"
	"skeji/pkg/logger"
	"skeji/pkg/model"
	"strings"

	"github.com/go-playground/validator/v10"
)

var (
	phoneRegex    = regexp.MustCompile(`^(?:|\+[1-9]\d{7,14})$`)
	validTimeZone = regexp.MustCompile(`^[A-Za-z0-9_\-/]+$`)
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
	if err := v.RegisterValidation("valid_url", validateUrl); err != nil {
		log.Fatal("Failed to register 'valid_url' validator",
			"error", err,
		)
	}

	if err := v.RegisterValidation("valid_phone", validPhoneNumber); err != nil {
		log.Fatal("Failed to register 'valid_phone' validator",
			"error", err,
		)
	}

	if err := v.RegisterValidation("participants_map", validateParticipantsMap); err != nil {
		log.Fatal("Failed to register 'participants_map' validator",
			"error", err,
		)
	}

	if err := v.RegisterValidation("maintainers_map", validateMaintainersMap); err != nil {
		log.Fatal("Failed to register 'maintainers_map' validator",
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

func validateSupportedTimeZone(tz string) bool {
	if !validTimeZone.MatchString(tz) {
		return false
	}
	if !locale.SupportedTimeZones[tz] {
		return false
	}
	return true
}

func validateParticipantsMap(fl validator.FieldLevel) bool {
	return validateMap(fl, func(key string) bool { return true }, phoneRegex.MatchString)
}

func validateMaintainersMap(fl validator.FieldLevel) bool {
	return validateMap(fl, phoneRegex.MatchString, func(key string) bool { return true })
}

func validateMap(fl validator.FieldLevel, keyValidator func(string) bool, valValidator func(string) bool) bool {
	value := fl.Field()
	if value.IsNil() {
		return true
	}
	mp, ok := value.Interface().(map[string]string)
	if !ok {
		return false
	}

	n := len(mp)
	if n == 0 {
		return true
	}
	if n < 1 || n > 200 {
		return false
	}
	for k, v := range mp {
		if k == "" || v == "" || !keyValidator(k) || !valValidator(v) {
			return false
		}
	}
	return true
}

func validateUrl(fl validator.FieldLevel) bool {
	input := strings.TrimSpace(fl.Field().String())

	if input == "" {
		return true
	}

	u, err := url.ParseRequestURI(input)
	if err != nil {
		return false
	}

	if u.Scheme != "https" {
		return false
	}

	if u.Host == "" {
		return false
	}

	hostname := u.Hostname()
	if !strings.Contains(hostname, ".") {
		return false
	}

	ip := net.ParseIP(hostname)
	if ip != nil && (ip.IsLoopback() || ip.IsPrivate()) {
		return false
	}

	if strings.Contains(u.Path, "..") {
		return false
	}

	return true
}

func validPhoneNumber(fl validator.FieldLevel) bool {
	phone := fl.Field().String()
	return phoneRegex.MatchString(phone)
}

func (v *BusinessUnitValidator) Validate(bu *model.BusinessUnit) error {
	if err := v.validate.Struct(bu); err != nil {
		var validationErrs validator.ValidationErrors
		if errors.As(err, &validationErrs) {
			return v.translateValidationErrors(validationErrs)
		}
		return err
	}
	if len(bu.Maintainers) > config.DefaultMaxMaintainersPerBusiness {
		return ValidationErrors{
			ValidationError{
				Field:   "Maintainers",
				Message: fmt.Sprintf("Maintainers count (%d) exceeds capacity (%d)", len(bu.Maintainers), config.DefaultMaxMaintainersPerBusiness),
			},
		}
	}

	if len(bu.Cities) < 1 || len(bu.Cities) > config.MaxCitiesForBusiness {
		return ValidationErrors{
			ValidationError{
				Field:   "Cities",
				Message: fmt.Sprintf("Cities count (%d) exceeds capacity (%d)", len(bu.Cities), config.MaxCitiesForBusiness),
			},
		}
	}

	if len(bu.Labels) > config.MaxLabelsForBusiness {
		return ValidationError{
			Field:   "Lables",
			Message: fmt.Sprintf("Labels count (%d) exceeds capacity (%d)", len(bu.Labels), config.MaxLabelsForBusiness),
		}
	}

	if !phoneRegex.MatchString(bu.AdminPhone) {
		return ValidationError{
			Field:   "AdminPhone",
			Message: fmt.Sprintf("AdminPhone must be a valid phone (%s)", bu.AdminPhone),
		}
	}

	if !validateSupportedTimeZone(bu.TimeZone) {
		return ValidationError{
			Field:   "TimeZone",
			Message: fmt.Sprintf("TimeZone '%s' is not supported", bu.TimeZone),
		}
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
		case "valid_url":
			message = fmt.Sprintf("invalid URL '%s', must be a valid HTTPS URL with a domain (e.g., https://example.com)", err.Value())
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
