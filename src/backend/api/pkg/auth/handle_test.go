package auth

import (
	"testing"
)

func TestValidatePilotHandle(t *testing.T) {
	tests := []struct {
		handle  string
		wantErr bool
		desc    string
	}{
		{"fpv", false, "valid minimum length (3)"},
		{"rotor_ninja", false, "valid handle with underscore"},
		{"pilot_42", false, "valid handle ending with number"},
		{"abcdefghijklmnopqrst", false, "valid maximum length (20)"},

		// Length violations
		{"ab", true, "too short (< 3)"},
		{"abcdefghijklmnopqrstu", true, "too long (> 20)"},

		// Boundary / character violations
		{"1pilot", true, "starts with number"},
		{"_pilot", true, "starts with underscore"},
		{"pilot_", true, "ends with underscore"},
		{"pilot-apex", true, "contains hyphen"},
		{"pilot.apex", true, "contains dot"},
		{"Pilot_Apex", true, "contains uppercase letters"},

		// Underscore repetition
		{"rotor__ninja", true, "consecutive underscores"},
		{"rotor___ninja", true, "three consecutive underscores"},

		// Forbidden substrings
		{"quadsmith", true, "contains quadsmith"},
		{"quad_smith", true, "contains quad_smith"},
		{"my_quadsmith", true, "contains quadsmith in middle"},
		{"quad_smith_fpv", true, "contains quad_smith in prefix"},

		// Reserved names
		{"admin", true, "equals admin"},
		{"root", true, "equals root"},
		{"admin_fpv", false, "allowed prefix not equal to admin"},
		{"root_pilot", false, "allowed prefix not equal to root"},
	}

	for _, tt := range tests {
		t.Run(tt.handle, func(t *testing.T) {
			err := ValidatePilotHandle(tt.handle)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePilotHandle(%q) error = %v, wantErr = %v (%s)", tt.handle, err, tt.wantErr, tt.desc)
			}
		})
	}
}
