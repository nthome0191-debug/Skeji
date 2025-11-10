package sanitizer

import (
	"reflect"
	"testing"
)

func TestNormalizeCities(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  []string
	}{
		{
			name:  "convert to lowercase",
			input: []string{"Tel Aviv", "JERUSALEM"},
			want:  []string{"telaviv", "jerusalem"},
		},
		{
			name:  "remove spaces",
			input: []string{"Tel Aviv", "Bnei Brak"},
			want:  []string{"telaviv", "bneibrak"},
		},
		{
			name:  "trim whitespace",
			input: []string{" Tel Aviv ", "  Jerusalem  "},
			want:  []string{"telaviv", "jerusalem"},
		},
		{
			name:  "remove duplicates",
			input: []string{"Tel Aviv", "tel aviv", "TELAVIV"},
			want:  []string{"telaviv"},
		},
		{
			name:  "filter empty strings",
			input: []string{"Tel Aviv", "", "  ", "Jerusalem"},
			want:  []string{"telaviv", "jerusalem"},
		},
		{
			name:  "empty input",
			input: []string{},
			want:  []string{},
		},
		{
			name:  "nil input",
			input: nil,
			want:  []string{},
		},
		{
			name:  "special characters",
			input: []string{"Kfar-Saba", "Ma'ale Adumim"},
			want:  []string{"kfarsaba", "maaleadumim"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeCities(tt.input)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NormalizeCities(%v) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestNormalizeLabels(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  []string
	}{
		{
			name:  "convert to lowercase",
			input: []string{"Haircut", "MASSAGE"},
			want:  []string{"haircut", "massage"},
		},
		{
			name:  "remove spaces",
			input: []string{"Hair Cut", "Body Massage"},
			want:  []string{"haircut", "bodymassage"},
		},
		{
			name:  "trim whitespace",
			input: []string{" Haircut ", "  Massage  "},
			want:  []string{"haircut", "massage"},
		},
		{
			name:  "remove duplicates",
			input: []string{"Haircut", "haircut", "HAIRCUT"},
			want:  []string{"haircut"},
		},
		{
			name:  "filter empty strings",
			input: []string{"Haircut", "", "  ", "Massage"},
			want:  []string{"haircut", "massage"},
		},
		{
			name:  "empty input",
			input: []string{},
			want:  []string{},
		},
		{
			name:  "nil input",
			input: nil,
			want:  []string{},
		},
		{
			name:  "special characters",
			input: []string{"Hair & Styling", "Spa-Treatment"},
			want:  []string{"hairstyling", "spatreatment"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeLabels(tt.input)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NormalizeLabels(%v) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
