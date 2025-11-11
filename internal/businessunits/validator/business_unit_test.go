package validator

import (
	"skeji/pkg/logger"
	"skeji/pkg/model"
	"strings"
	"testing"
)

func TestValidateSupportedCountry(t *testing.T) {
	log := logger.New(logger.Config{
		Level:     "info",
		Format:    logger.JSON,
		AddSource: false,
		Service:   "test",
	})
	validator := NewBusinessUnitValidator(log)

	tests := []struct {
		name      string
		bu        *model.BusinessUnit
		wantError bool
	}{
		{
			name: "valid Israel phone",
			bu: &model.BusinessUnit{
				Name:       "Test Business",
				Cities:     []string{"Tel Aviv"},
				Labels:     []string{"Haircut"},
				AdminPhone: "+972541234567",
			},
			wantError: false,
		},
		{
			name: "valid US phone",
			bu: &model.BusinessUnit{
				Name:       "Test Business",
				Cities:     []string{"New York"},
				Labels:     []string{"Haircut"},
				AdminPhone: "+12125551234",
			},
			wantError: false,
		},
		{
			name: "unsupported country",
			bu: &model.BusinessUnit{
				Name:       "Test Business",
				Cities:     []string{"London"},
				Labels:     []string{"Haircut"},
				AdminPhone: "+442071234567",
			},
			wantError: true,
		},
		{
			name: "invalid format",
			bu: &model.BusinessUnit{
				Name:       "Test Business",
				Cities:     []string{"Tel Aviv"},
				Labels:     []string{"Haircut"},
				AdminPhone: "not-a-phone",
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(tt.bu)
			if (err != nil) != tt.wantError {
				t.Errorf("Validate() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestValidateURL(t *testing.T) {
	log := logger.New(logger.Config{
		Level:     "info",
		Format:    logger.JSON,
		AddSource: false,
		Service:   "test",
	})
	validator := NewBusinessUnitValidator(log)

	tests := []struct {
		name        string
		websiteURLs []string
		wantError   bool
	}{
		{
			name:        "valid https URL",
			websiteURLs: []string{"https://example.com"},
			wantError:   false,
		},
		{
			name:        "valid https with path",
			websiteURLs: []string{"https://example.com/path/to/page"},
			wantError:   false,
		},
		{
			name:        "multiple valid URLs",
			websiteURLs: []string{"https://example.com", "https://facebook.com/page", "https://instagram.com/profile"},
			wantError:   false,
		},
		{
			name:        "exactly 5 URLs (max allowed)",
			websiteURLs: []string{"https://example.com", "https://facebook.com", "https://instagram.com", "https://twitter.com", "https://linkedin.com"},
			wantError:   false,
		},
		{
			name:        "more than 5 URLs",
			websiteURLs: []string{"https://example.com", "https://facebook.com", "https://instagram.com", "https://twitter.com", "https://linkedin.com", "https://youtube.com"},
			wantError:   true,
		},
		{
			name:        "empty array allowed",
			websiteURLs: []string{},
			wantError:   false,
		},
		{
			name:        "http not allowed",
			websiteURLs: []string{"http://example.com"},
			wantError:   true,
		},
		{
			name:        "no scheme",
			websiteURLs: []string{"example.com"},
			wantError:   true,
		},
		{
			name:        "localhost not allowed",
			websiteURLs: []string{"https://localhost:8080"},
			wantError:   true,
		},
		{
			name:        "private IP not allowed",
			websiteURLs: []string{"https://192.168.1.1"},
			wantError:   true,
		},
		{
			name:        "path traversal not allowed",
			websiteURLs: []string{"https://example.com/../etc/passwd"},
			wantError:   true,
		},
		{
			name:        "one valid and one invalid",
			websiteURLs: []string{"https://example.com", "http://invalid.com"},
			wantError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bu := &model.BusinessUnit{
				Name:        "Test Business",
				Cities:      []string{"Tel Aviv"},
				Labels:      []string{"Haircut"},
				AdminPhone:  "+972541234567",
				WebsiteURLs: tt.websiteURLs,
			}
			err := validator.Validate(bu)
			if (err != nil) != tt.wantError {
				t.Errorf("Validate() with URLs %v error = %v, wantError %v", tt.websiteURLs, err, tt.wantError)
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
	validator := NewBusinessUnitValidator(log)

	tests := []struct {
		name      string
		bu        *model.BusinessUnit
		wantError bool
		errorMsg  string
	}{
		{
			name: "missing name",
			bu: &model.BusinessUnit{
				Cities:     []string{"Tel Aviv"},
				Labels:     []string{"Haircut"},
				AdminPhone: "+972541234567",
			},
			wantError: true,
			errorMsg:  "name",
		},
		{
			name: "missing cities",
			bu: &model.BusinessUnit{
				Name:       "Test Business",
				Labels:     []string{"Haircut"},
				AdminPhone: "+972541234567",
			},
			wantError: true,
			errorMsg:  "cities",
		},
		{
			name: "empty cities",
			bu: &model.BusinessUnit{
				Name:       "Test Business",
				Cities:     []string{},
				Labels:     []string{"Haircut"},
				AdminPhone: "+972541234567",
			},
			wantError: true,
			errorMsg:  "cities",
		},
		{
			name: "missing labels",
			bu: &model.BusinessUnit{
				Name:       "Test Business",
				Cities:     []string{"Tel Aviv"},
				AdminPhone: "+972541234567",
			},
			wantError: true,
			errorMsg:  "labels",
		},
		{
			name: "empty labels",
			bu: &model.BusinessUnit{
				Name:       "Test Business",
				Cities:     []string{"Tel Aviv"},
				Labels:     []string{},
				AdminPhone: "+972541234567",
			},
			wantError: true,
			errorMsg:  "labels",
		},
		{
			name: "missing admin phone",
			bu: &model.BusinessUnit{
				Name:   "Test Business",
				Cities: []string{"Tel Aviv"},
				Labels: []string{"Haircut"},
			},
			wantError: true,
			errorMsg:  "admin_phone",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(tt.bu)
			if (err != nil) != tt.wantError {
				t.Errorf("Validate() error = %v, wantError %v", err, tt.wantError)
			}
			if tt.wantError && err != nil {
				errStr := err.Error()
				// Field names in errors are capitalized (e.g., "AdminPhone" not "admin_phone")
				expectedMsg := strings.ReplaceAll(tt.errorMsg, "_", "")
				if !strings.Contains(strings.ToLower(errStr), strings.ToLower(expectedMsg)) {
					t.Errorf("Expected error to contain %q (checking for %q), got %q", tt.errorMsg, expectedMsg, err.Error())
				}
			}
		})
	}
}

func TestValidateArraySizes(t *testing.T) {
	log := logger.New(logger.Config{
		Level:     "info",
		Format:    logger.JSON,
		AddSource: false,
		Service:   "test",
	})
	validator := NewBusinessUnitValidator(log)

	tests := []struct {
		name      string
		cities    []string
		labels    []string
		wantError bool
	}{
		{
			name:      "max cities exceeded",
			cities:    make([]string, 51),
			labels:    []string{"Haircut"},
			wantError: true,
		},
		{
			name:      "max labels exceeded",
			cities:    []string{"Tel Aviv"},
			labels:    make([]string, 11),
			wantError: true,
		},
		{
			name:      "exactly at max cities",
			cities:    make([]string, 50),
			labels:    []string{"Haircut"},
			wantError: false,
		},
		{
			name:      "exactly at max labels",
			cities:    []string{"Tel Aviv"},
			labels:    make([]string, 10),
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Fill arrays with valid values
			for i := range tt.cities {
				tt.cities[i] = "City"
			}
			for i := range tt.labels {
				tt.labels[i] = "Label"
			}

			bu := &model.BusinessUnit{
				Name:       "Test Business",
				Cities:     tt.cities,
				Labels:     tt.labels,
				AdminPhone: "+972541234567",
			}
			err := validator.Validate(bu)
			if (err != nil) != tt.wantError {
				t.Errorf("Validate() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestValidateNameLength(t *testing.T) {
	log := logger.New(logger.Config{
		Level:     "info",
		Format:    logger.JSON,
		AddSource: false,
		Service:   "test",
	})
	validator := NewBusinessUnitValidator(log)

	tests := []struct {
		name      string
		bizName   string
		wantError bool
	}{
		{
			name:      "too short (1 char)",
			bizName:   "A",
			wantError: true,
		},
		{
			name:      "minimum length (2 chars)",
			bizName:   "AB",
			wantError: false,
		},
		{
			name:      "maximum length (100 chars)",
			bizName:   strings.Repeat("A", 100),
			wantError: false,
		},
		{
			name:      "too long (101 chars)",
			bizName:   strings.Repeat("A", 101),
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bu := &model.BusinessUnit{
				Name:       tt.bizName,
				Cities:     []string{"Tel Aviv"},
				Labels:     []string{"Haircut"},
				AdminPhone: "+972541234567",
			}
			err := validator.Validate(bu)
			if (err != nil) != tt.wantError {
				t.Errorf("Validate() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestValidateTimezone(t *testing.T) {
	log := logger.New(logger.Config{
		Level:     "info",
		Format:    logger.JSON,
		AddSource: false,
		Service:   "test",
	})
	validator := NewBusinessUnitValidator(log)

	tests := []struct {
		name      string
		timezone  string
		wantError bool
	}{
		{
			name:      "valid timezone",
			timezone:  "Asia/Jerusalem",
			wantError: false,
		},
		{
			name:      "valid timezone US",
			timezone:  "America/New_York",
			wantError: false,
		},
		{
			name:      "UTC",
			timezone:  "UTC",
			wantError: false,
		},
		{
			name:      "invalid timezone",
			timezone:  "Invalid/Timezone",
			wantError: true,
		},
		{
			name:      "empty timezone is optional",
			timezone:  "",
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bu := &model.BusinessUnit{
				Name:       "Test Business",
				Cities:     []string{"Tel Aviv"},
				Labels:     []string{"Haircut"},
				AdminPhone: "+972541234567",
				TimeZone:   tt.timezone,
			}
			err := validator.Validate(bu)
			if (err != nil) != tt.wantError {
				t.Errorf("Validate() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}
