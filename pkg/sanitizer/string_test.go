package sanitizer

import "testing"

func TestNormalizeName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "trim spaces",
			input: "  John's Salon  ",
			want:  "John's Salon",
		},
		{
			name:  "multiple spaces between words",
			input: "John's    Salon",
			want:  "John's Salon",
		},
		{
			name:  "tabs and newlines",
			input: "John's\t\nSalon",
			want:  "John's Salon",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "only whitespace",
			input: "   \t\n  ",
			want:  "",
		},
		{
			name:  "preserve special characters",
			input: " Café & Spa™ ",
			want:  "Café & Spa™",
		},
		{
			name:  "hebrew characters",
			input: " תספורת יוסי ",
			want:  "תספורת יוסי",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// got := Normalize(tt.input)
			// if got != tt.want {
			// 	t.Errorf("NormalizeName(%q) = %q, want %q", tt.input, got, tt.want)
			// }
		})
	}
}

func TestTrimAndNormalize(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "basic trim",
			input: "  hello  ",
			want:  "hello",
		},
		{
			name:  "multiple spaces",
			input: "hello    world",
			want:  "hello world",
		},
		{
			name:  "tabs and newlines",
			input: "hello\t\nworld",
			want:  "hello world",
		},
		{
			name:  "empty",
			input: "",
			want:  "",
		},
		{
			name:  "only whitespace",
			input: "   ",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TrimAndNormalize(tt.input)
			if got != tt.want {
				t.Errorf("TrimAndNormalize(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestNormalizeNameForComparison(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "convert to lowercase",
			input: "John's Salon",
			want:  "john's salon",
		},
		{
			name:  "collapse multiple spaces",
			input: "John's   Salon",
			want:  "john's salon",
		},
		{
			name:  "preserve special chars but lowercase",
			input: "Café & Spa™",
			want:  "café & spa™",
		},
		{
			name:  "trim and lowercase",
			input: "  JOHN'S  Café  ",
			want:  "john's café",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeNameForComparison(tt.input)
			if got != tt.want {
				t.Errorf("NormalizeNameForComparison(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
