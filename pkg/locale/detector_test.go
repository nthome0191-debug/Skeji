package locale

import (
	"testing"
)

func TestInferCountryFromPhone(t *testing.T) {
	tests := []struct {
		name       string
		phone      string
		wantCode   string
		wantNil    bool
	}{
		{
			name:     "Israel phone",
			phone:    "+972541234567",
			wantCode: "IL",
			wantNil:  false,
		},
		{
			name:     "Israel phone without plus",
			phone:    "972541234567",
			wantCode: "IL",
			wantNil:  false,
		},
		{
			name:     "US phone",
			phone:    "+12125551234",
			wantCode: "US",
			wantNil:  false,
		},
		{
			name:     "US phone without plus",
			phone:    "12125551234",
			wantCode: "US",
			wantNil:  false,
		},
		{
			name:    "unknown country",
			phone:   "+442071234567",
			wantNil: true,
		},
		{
			name:    "empty phone",
			phone:   "",
			wantNil: true,
		},
		{
			name:    "invalid phone",
			phone:   "not-a-phone",
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := InferCountryFromPhone(tt.phone)
			if tt.wantNil {
				if got != nil {
					t.Errorf("InferCountryFromPhone(%q) = %v, want nil", tt.phone, got)
				}
			} else {
				if got == nil {
					t.Errorf("InferCountryFromPhone(%q) = nil, want country with code %q", tt.phone, tt.wantCode)
				} else if got.Code != tt.wantCode {
					t.Errorf("InferCountryFromPhone(%q).Code = %q, want %q", tt.phone, got.Code, tt.wantCode)
				}
			}
		})
	}
}

func TestInferTimezoneFromPhone(t *testing.T) {
	tests := []struct {
		name  string
		phone string
		want  string
	}{
		{
			name:  "Israel phone returns Jerusalem timezone",
			phone: "+972541234567",
			want:  "Asia/Jerusalem",
		},
		{
			name:  "US phone returns New York timezone",
			phone: "+12125551234",
			want:  "America/New_York",
		},
		{
			name:  "unknown phone returns UTC",
			phone: "+442071234567",
			want:  "UTC",
		},
		{
			name:  "empty phone returns UTC",
			phone: "",
			want:  "UTC",
		},
		{
			name:  "invalid phone returns UTC",
			phone: "invalid",
			want:  "UTC",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := InferTimezoneFromPhone(tt.phone)
			if got != tt.want {
				t.Errorf("InferTimezoneFromPhone(%q) = %q, want %q", tt.phone, got, tt.want)
			}
		})
	}
}

func TestDetectRegion(t *testing.T) {
	tests := []struct {
		name     string
		timezone string
		want     string
	}{
		{
			name:     "Jerusalem timezone",
			timezone: "Asia/Jerusalem",
			want:     "IL",
		},
		{
			name:     "Tel Aviv timezone (if exists)",
			timezone: "Asia/Tel_Aviv",
			want:     "IL",
		},
		{
			name:     "New York timezone",
			timezone: "America/New_York",
			want:     "US",
		},
		{
			name:     "Los Angeles timezone",
			timezone: "America/Los_Angeles",
			want:     "US",
		},
		{
			name:     "Chicago timezone defaults to IL (not in map)",
			timezone: "America/Chicago",
			want:     "IL",
		},
		{
			name:     "UTC defaults to IL",
			timezone: "UTC",
			want:     "IL",
		},
		{
			name:     "London timezone defaults to IL",
			timezone: "Europe/London",
			want:     "IL",
		},
		{
			name:     "empty timezone defaults to IL",
			timezone: "",
			want:     "IL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectRegion(tt.timezone)
			if got != tt.want {
				t.Errorf("DetectRegion(%q) = %q, want %q", tt.timezone, got, tt.want)
			}
		})
	}
}
