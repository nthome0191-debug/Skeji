package sanitizer

import "testing"

func TestNormalizePhone(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "valid E.164 format",
			input: "+972541234567",
			want:  "+972541234567",
		},
		{
			name:  "with spaces",
			input: "+972 54 123 4567",
			want:  "+972541234567",
		},
		{
			name:  "with dashes",
			input: "+972-54-123-4567",
			want:  "+972541234567",
		},
		{
			name:  "with parentheses",
			input: "+1 (212) 555-1234",
			want:  "+12125551234",
		},
		{
			name:  "leading and trailing spaces",
			input: "  +972541234567  ",
			want:  "+972541234567",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "only whitespace",
			input: "   ",
			want:  "",
		},
		{
			name:  "no plus sign",
			input: "972541234567",
			want:  "+972541234567",
		},
		{
			name:  "mixed special chars",
			input: " +972-54.123 4567 ",
			want:  "+972541234567",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizePhone(tt.input)
			if got != tt.want {
				t.Errorf("NormalizePhone(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// func TestNormalizeMaintainers(t *testing.T) {
// 	tests := []struct {
// 		name  string
// 		input []string
// 		want  []string
// 	}{
// 		{
// 			name:  "valid phones",
// 			input: []string{"+972541234567", "+12125551234"},
// 			want:  []string{"+972541234567", "+12125551234"},
// 		},
// 		{
// 			name:  "phones with spaces",
// 			input: []string{"+972 54 123 4567", "+1 212 555 1234"},
// 			want:  []string{"+972541234567", "+12125551234"},
// 		},
// 		{
// 			name:  "mixed valid and invalid",
// 			input: []string{"+972541234567", "invalid", "+12125551234"},
// 			want:  []string{"+972541234567", "+12125551234"},
// 		},
// 		{
// 			name:  "empty strings filtered",
// 			input: []string{"+972541234567", "", "+12125551234"},
// 			want:  []string{"+972541234567", "+12125551234"},
// 		},
// 		{
// 			name:  "duplicates removed",
// 			input: []string{"+972541234567", "+972541234567", "+12125551234"},
// 			want:  []string{"+972541234567", "+12125551234"},
// 		},
// 		{
// 			name:  "empty input",
// 			input: []string{},
// 			want:  []string{},
// 		},
// 		{
// 			name:  "nil input",
// 			input: nil,
// 			want:  []string{},
// 		},
// 		{
// 			name:  "all invalid",
// 			input: []string{"invalid", "", "bad"},
// 			want:  []string{},
// 		},
// 	}

// for _, tt := range tests {
// 	t.Run(tt.name, func(t *testing.T) {
// 		got := NormalizeMaintainers(tt.input)
// 		if len(got) != len(tt.want) {
// 			t.Errorf("NormalizeMaintainers(%v) length = %d, want %d", tt.input, len(got), len(tt.want))
// 			return
// 		}
// 		for i := range got {
// 			if got[i] != tt.want[i] {
// 				t.Errorf("NormalizeMaintainers(%v)[%d] = %q, want %q", tt.input, i, got[i], tt.want[i])
// 			}
// 		}
// 	})
// }
// }
