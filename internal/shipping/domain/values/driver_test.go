package values_test

import (
	"testing"

	"last-mile-task/internal/shipping/domain/values"
)

func TestDriverName_FullName(t *testing.T) {
	tests := []struct {
		name      string
		firstName string
		middle    string
		last      string
		expected  string
	}{
		{
			name:      "all three names",
			firstName: "Ivan",
			middle:    "Ivanovich",
			last:      "Ivanov",
			expected:  "Ivan Ivanovich Ivanov",
		},
		{
			name:      "first name only",
			firstName: "Ivan",
			middle:    "",
			last:      "",
			expected:  "Ivan",
		},
		{
			name:      "first and last name",
			firstName: "Ivan",
			middle:    "",
			last:      "Ivanov",
			expected:  "Ivan Ivanov",
		},
		{
			name:      "first and middle name",
			firstName: "Ivan",
			middle:    "Ivanovich",
			last:      "",
			expected:  "Ivan Ivanovich",
		},
		{
			name:      "all empty",
			firstName: "",
			middle:    "",
			last:      "",
			expected:  "",
		},
		{
			name:      "only last name",
			firstName: "",
			middle:    "",
			last:      "Ivanov",
			expected:  " Ivanov",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dn := values.NewDriverName(tt.firstName, tt.middle, tt.last)
			got := dn.FullName()
			if got != tt.expected {
				t.Fatalf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}
