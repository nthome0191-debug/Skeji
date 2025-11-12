package sanitizer

import (
	"skeji/pkg/config"
	"testing"
)

func TestNormalizePhone_EdgeCases(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		actualResult  string // What NormalizePhone actually returns
		idealBehavior string // What it SHOULD return
		documentsBug  bool
	}{
		{
			name:          "invalid phone with letters becomes israeli number",
			input:         "invalid-phone-123",
			actualResult:  "+972123", // BUG: Adds Israeli country code to invalid input
			idealBehavior: "should return empty or error for invalid input",
			documentsBug:  true,
		},
		{
			name:          "too short phone becomes empty",
			input:         "+1",
			actualResult:  "", // Returns empty for too short input
			idealBehavior: "should preserve as-is or return validation error",
			documentsBug:  true,
		},
		{
			name:          "only special characters become empty",
			input:         "()---   ",
			actualResult:  "", // All chars stripped, results in empty
			idealBehavior: "should return validation error",
			documentsBug:  true,
		},
		{
			name:          "mixed invalid chars treated as israeli number",
			input:         "abc-123-def",
			actualResult:  "+972123333", // BUG: Letters become "333", adds IL code
			idealBehavior: "should return empty or error for non-digit input",
			documentsBug:  true,
		},
		{
			name:          "extremely long phone becomes empty",
			input:         "+1234567890123456789012345678901234567890",
			actualResult:  "", // Too long, becomes empty
			idealBehavior: "should return validation error or truncate with warning",
			documentsBug:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizePhone(tt.input)
			if result != tt.actualResult {
				t.Errorf("NormalizePhone(%q) = %q, expected current behavior to be %q", tt.input, result, tt.actualResult)
			}

			if tt.documentsBug {
				t.Logf("DOCUMENTED BUG: Input %q -> %q (ideal: %s)", tt.input, result, tt.idealBehavior)
			}
		})
	}
}

func TestNormalizeMaintainers_EmptyResults(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "all invalid phones filtered out",
			input:    []string{"invalid1", "invalid2", "invalid3"},
			expected: []string{},
		},
		{
			name:     "mixed valid and many invalid",
			input:    []string{"invalid1", "+972541234567", "invalid2", "invalid3"},
			expected: []string{"+972541234567"},
		},
		{
			name:     "empty strings filtered",
			input:    []string{"", "", ""},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeMaintainers(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("expected %d results, got %d", len(tt.expected), len(result))
			}
		})
	}
}

func TestNormalizePriority_ConfigEdgeCases(t *testing.T) {
	tests := []struct {
		name                 string
		minPriority          int
		maxPriority          int
		inputPriority        int64
		expectedPriority     int64
		expectsNormalization bool
	}{
		{
			name:                 "below minimum",
			minPriority:          1,
			maxPriority:          100,
			inputPriority:        -10,
			expectedPriority:     1,
			expectsNormalization: true,
		},
		{
			name:                 "above maximum",
			minPriority:          1,
			maxPriority:          100,
			inputPriority:        200,
			expectedPriority:     100,
			expectsNormalization: true,
		},
		{
			name:                 "within range",
			minPriority:          1,
			maxPriority:          100,
			inputPriority:        50,
			expectedPriority:     50,
			expectsNormalization: false,
		},
		{
			name:                 "exactly at minimum",
			minPriority:          1,
			maxPriority:          100,
			inputPriority:        1,
			expectedPriority:     1,
			expectsNormalization: false,
		},
		{
			name:                 "exactly at maximum",
			minPriority:          1,
			maxPriority:          100,
			inputPriority:        100,
			expectedPriority:     100,
			expectsNormalization: false,
		},
		{
			name:                 "BUG TEST: min > max config",
			minPriority:          100,
			maxPriority:          1,
			inputPriority:        50,
			expectedPriority:     100, // Will clamp to min first
			expectsNormalization: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				MinBusinessPriority: tt.minPriority,
				MaxBusinessPriority: tt.maxPriority,
			}

			result := NormalizePriority(cfg, tt.inputPriority)

			if result != tt.expectedPriority {
				t.Errorf("expected %d, got %d", tt.expectedPriority, result)
			}

			// Document configuration validation issue
			if tt.minPriority > tt.maxPriority {
				t.Logf("WARNING: Configuration has min > max (%d > %d) - no validation!",
					tt.minPriority, tt.maxPriority)
			}
		})
	}
}

func TestNormalizeCities_SpecialCharacters(t *testing.T) {
	tests := []struct {
		name   string
		input  []string
		output []string
	}{
		{
			name:   "SQL injection attempt",
			input:  []string{"Tel Aviv'; DROP TABLE cities;--"},
			output: []string{"telavivdroptablecities"},
		},
		{
			name:   "Script injection",
			input:  []string{"<script>alert('xss')</script>"},
			output: []string{"scriptalertxssscript"},
		},
		{
			name:   "Unicode normalization - inconsistent handling",
			input:  []string{"Café", "Cafe\u0301"},
			output: []string{"café", "cafe"}, // BUG: "Café" keeps accent, "Cafe\u0301" loses it
		},
		{
			name:   "emoji in city name",
			input:  []string{"Tel Aviv 🏙️"},
			output: []string{"telaviv"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeCities(tt.input)
			if len(result) != len(tt.output) {
				t.Errorf("expected %d results, got %d", len(tt.output), len(result))
			}
			for i, expected := range tt.output {
				if i < len(result) && result[i] != expected {
					t.Errorf("index %d: expected %q, got %q", i, expected, result[i])
				}
			}
		})
	}
}

func TestNormalizeLabels_Duplicates(t *testing.T) {
	input := []string{
		"Haircut",
		"HAIRCUT",
		"haircut",
		"Hair Cut",
		"hair-cut",
	}

	result := NormalizeLabels(input)

	// All should normalize to the same value and be deduplicated
	if len(result) != 1 {
		t.Errorf("expected 1 unique label, got %d: %v", len(result), result)
	}

	if len(result) > 0 && result[0] != "haircut" {
		t.Errorf("expected 'haircut', got %q", result[0])
	}
}

func TestNormalizeName_ExtremelyLongInput(t *testing.T) {
	// Test with very long input
	longName := ""
	for i := 0; i < 10000; i++ {
		longName += "a "
	}

	result := Normalize(longName)

	// Should not panic
	if result == "" {
		t.Error("expected non-empty result for long input")
	}

	// Should normalize spaces
	if len(result) >= len(longName) {
		t.Error("expected space normalization to reduce length")
	}
}
