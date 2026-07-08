package kyc

import "testing"

func TestLimitsMonotonic(t *testing.T) {
	l0 := LimitsFor(Level0)
	l1 := LimitsFor(Level1)
	l2 := LimitsFor(Level2)
	if l0.DailyMinor >= l1.DailyMinor || l1.DailyMinor >= l2.DailyMinor {
		t.Fatal("daily limits must increase with level")
	}
	if l0.MonthlyMinor >= l1.MonthlyMinor || l1.MonthlyMinor >= l2.MonthlyMinor {
		t.Fatal("monthly limits must increase with level")
	}
}

func TestUnknownLevelFallsBackToStrictest(t *testing.T) {
	if LimitsFor(Level(99)) != LimitsFor(Level0) {
		t.Fatal("unknown level must fall back to the strictest limits")
	}
	if Level(99).Valid() {
		t.Fatal("level 99 must be invalid")
	}
}
