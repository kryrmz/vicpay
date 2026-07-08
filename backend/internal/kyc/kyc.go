// Package kyc models VicPay's progressive-onboarding tiers. Every new user
// starts at Level 0 (frictionless: phone + password only) with the tightest
// limits; higher levels unlock larger limits after identity verification. Unlike
// KiramoPay, a new user is pinned to the real Level 0 limits from registration,
// never a more permissive wallet default.
package kyc

// Level is a KYC tier from 0 (basic) to 2 (complete).
type Level int

const (
	// Level0 is available with no verification beyond a real phone number.
	Level0 Level = 0
	// Level1 requires a verified government ID.
	Level1 Level = 1
	// Level2 additionally requires proof of address.
	Level2 Level = 2
)

// Limits are per-user transaction ceilings in minor units, per currency bucket.
// Values below are expressed in USD minor units (cents) as a reference bucket;
// production wires per-currency tables.
type Limits struct {
	DailyMinor   int64
	MonthlyMinor int64
}

// limitsByLevel is the reference limit table.
var limitsByLevel = map[Level]Limits{
	Level0: {DailyMinor: 20_000, MonthlyMinor: 100_000},      // $200 / $1,000
	Level1: {DailyMinor: 100_000, MonthlyMinor: 1_000_000},   // $1,000 / $10,000
	Level2: {DailyMinor: 400_000, MonthlyMinor: 4_000_000},   // $4,000 / $40,000
}

// LimitsFor returns the limits for a level, defaulting to the strictest (L0) for
// any out-of-range value so an unknown level never grants more access.
func LimitsFor(l Level) Limits {
	if lim, ok := limitsByLevel[l]; ok {
		return lim
	}
	return limitsByLevel[Level0]
}

// Valid reports whether l is a known level.
func (l Level) Valid() bool {
	_, ok := limitsByLevel[l]
	return ok
}
