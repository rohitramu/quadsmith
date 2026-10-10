package auth

import (
	"math"
	"testing"
	"time"
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

func TestHandleReclaimCooldownLogic(t *testing.T) {
	now := time.Now()

	canReclaimArchived := func(lastRenamedAt time.Time, current time.Time) (bool, int) {
		elapsed := current.Sub(lastRenamedAt)
		if elapsed < 24*time.Hour {
			return true, 0 // 24h grace period
		}
		if elapsed >= 7*24*time.Hour {
			return true, 0 // 7-day cooldown elapsed
		}
		daysRemaining := int(math.Ceil((7*24*time.Hour - elapsed).Hours() / 24))
		return false, daysRemaining
	}

	canChangeToNewHandle := func(lastRenamedAt time.Time, current time.Time) (bool, int) {
		elapsed := current.Sub(lastRenamedAt)
		if elapsed >= 30*24*time.Hour {
			return true, 0
		}
		daysRemaining := int(math.Ceil((30*24*time.Hour - elapsed).Hours() / 24))
		return false, daysRemaining
	}

	// 1. Within 24 hours of rename: instant reclaim allowed
	canReclaim, days := canReclaimArchived(now.Add(-2*time.Hour), now)
	if !canReclaim || days != 0 {
		t.Errorf("expected instant reclaim allowed at 2 hours elapsed, got canReclaim=%v, days=%d", canReclaim, days)
	}

	canReclaim, days = canReclaimArchived(now.Add(-23*time.Hour), now)
	if !canReclaim || days != 0 {
		t.Errorf("expected instant reclaim allowed at 23 hours elapsed, got canReclaim=%v, days=%d", canReclaim, days)
	}

	// 2. Between 24h and 7 days: reclaim locked
	canReclaim, days = canReclaimArchived(now.Add(-25*time.Hour), now)
	if canReclaim || days != 6 {
		t.Errorf("expected reclaim locked at 25 hours elapsed, got canReclaim=%v, days=%d (expected 6)", canReclaim, days)
	}

	canReclaim, days = canReclaimArchived(now.Add(-3*24*time.Hour), now)
	if canReclaim || days != 4 {
		t.Errorf("expected reclaim locked at 3 days elapsed, got canReclaim=%v, days=%d (expected 4)", canReclaim, days)
	}

	// 3. After 7 days: reclaim allowed
	canReclaim, days = canReclaimArchived(now.Add(-7*24*time.Hour), now)
	if !canReclaim || days != 0 {
		t.Errorf("expected reclaim allowed at 7 days elapsed, got canReclaim=%v, days=%d", canReclaim, days)
	}

	canReclaim, days = canReclaimArchived(now.Add(-14*24*time.Hour), now)
	if !canReclaim || days != 0 {
		t.Errorf("expected reclaim allowed at 14 days elapsed, got canReclaim=%v, days=%d", canReclaim, days)
	}

	// 4. Brand-new handle cooldown (strictly 30 days)
	canNew, newDays := canChangeToNewHandle(now.Add(-10*24*time.Hour), now)
	if canNew || newDays != 20 {
		t.Errorf("expected brand-new handle locked at 10 days, got canNew=%v, days=%d (expected 20)", canNew, newDays)
	}

	canNew, newDays = canChangeToNewHandle(now.Add(-29*24*time.Hour), now)
	if canNew || newDays != 1 {
		t.Errorf("expected brand-new handle locked at 29 days, got canNew=%v, days=%d (expected 1)", canNew, newDays)
	}

	canNew, newDays = canChangeToNewHandle(now.Add(-30*24*time.Hour), now)
	if !canNew || newDays != 0 {
		t.Errorf("expected brand-new handle allowed at 30 days, got canNew=%v, days=%d", canNew, newDays)
	}
}

func TestHoldWindowAndWarningWorkerCriteria(t *testing.T) {
	now := time.Now()

	isEligibleFor7DayWarning := func(firstArchivedAt time.Time, current time.Time, warningSent bool) bool {
		if warningSent {
			return false
		}
		// 23 days elapsed <= current AND 30 days > current
		return !current.Before(firstArchivedAt.Add(23*24*time.Hour)) && current.Before(firstArchivedAt.Add(30*24*time.Hour))
	}

	isHoldActive := func(firstArchivedAt time.Time, current time.Time) bool {
		return current.Before(firstArchivedAt.Add(30 * 24 * time.Hour))
	}

	// 5 days elapsed: hold active, warning NOT eligible
	archived5d := now.Add(-5 * 24 * time.Hour)
	if !isHoldActive(archived5d, now) {
		t.Errorf("expected hold active at 5 days")
	}
	if isEligibleFor7DayWarning(archived5d, now, false) {
		t.Errorf("expected warning not eligible at 5 days")
	}

	// 23 days elapsed: hold active, warning IS eligible
	archived23d := now.Add(-23 * 24 * time.Hour)
	if !isHoldActive(archived23d, now) {
		t.Errorf("expected hold active at 23 days")
	}
	if !isEligibleFor7DayWarning(archived23d, now, false) {
		t.Errorf("expected warning eligible at 23 days (7 days remaining)")
	}
	// Once sent, should not be eligible again
	if isEligibleFor7DayWarning(archived23d, now, true) {
		t.Errorf("expected warning not eligible when already sent")
	}

	// 31 days elapsed: hold EXPIRED, warning NOT eligible
	archived31d := now.Add(-31 * 24 * time.Hour)
	if isHoldActive(archived31d, now) {
		t.Errorf("expected hold expired at 31 days")
	}
	if isEligibleFor7DayWarning(archived31d, now, false) {
		t.Errorf("expected warning not eligible when hold has expired")
	}
}
