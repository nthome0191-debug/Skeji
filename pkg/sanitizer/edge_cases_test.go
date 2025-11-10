package sanitizer

import (
	"skeji/pkg/config"
	"testing"
)

func TestNormalizePhone_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "returns empty for invalid phone",
			input:    "invalid-phone-123",
			expected: "",
		},
		{
			name:     "returns empty for too short",
			input:    "+1",
			expected: "+1",
		},
		{
			name:     "handles only special characters",
			input:    "()---   ",
			expected: "+",
		},
		{
			name:     "handles mixed invalid chars",
			input:    "abc-123-def",
			expected: "+123",
		},
		{
			name:     "extremely long input",
			input:    "+1234567890123456789012345678901234567890",
			expected: "+1234567890123456789012345678901234567890",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizePhone(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizePhone(%q) = %q, want %q", tt.input, result, tt.expected)
			}

			// Document the bug: empty results need error handling
			if tt.input != "" && result == "" {
				t.Logf("WARNING: Input %q normalized to empty string - needs error handling", tt.input)
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
		name               string
		minPriority        int
		maxPriority        int
		inputPriority      int64
		expectedPriority   int64
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
			name:   "Unicode normalization",
			input:  []string{"Café", "Cafe\u0301"},
			output: []string{"café", "café"},
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

	result := NormalizeName(longName)

	// Should not panic
	if result == "" {
		t.Error("expected non-empty result for long input")
	}

	// Should normalize spaces
	if len(result) >= len(longName) {
		t.Error("expected space normalization to reduce length")
	}
}
