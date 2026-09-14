package generator

import (
	"testing"
)

func TestGenerateRandomPassword(t *testing.T) {
	tests := []struct {
		name           string
		length         int
		expectedLength int
	}{
		{
			name:           "Generate password of length 8",
			length:         8,
			expectedLength: 8,
		},
		{
			name:           "Customized length 32",
			length:         32,
			expectedLength: 32,
		},
		{
			name:           "By Default length if  0 or negative",
			length:         0,
			expectedLength: 24,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			pwd, err := GenerateRandomPassword(test.length)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(pwd) != test.expectedLength {
				t.Errorf("expected %d, received %d", test.expectedLength, len(pwd))
			}
		})
	}
}
