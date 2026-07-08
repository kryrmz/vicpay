// Package validator holds small, dependency-free input validators. Phone rules
// are E.164-based and country-agnostic so VicPay is not tied to a single market;
// national-id validation is intentionally left to the KYC/onboarding layer.
package validator

import (
	"regexp"
	"strings"
)

var (
	emailRE = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	// E.164: leading '+', 8 to 15 digits, first digit non-zero.
	phoneRE = regexp.MustCompile(`^\+[1-9]\d{7,14}$`)
)

// Email reports whether s is a plausible email address.
func Email(s string) bool {
	s = strings.TrimSpace(s)
	return len(s) <= 254 && emailRE.MatchString(s)
}

// PhoneE164 reports whether s is a valid E.164 phone number (e.g. +50688881234).
func PhoneE164(s string) bool {
	return phoneRE.MatchString(strings.TrimSpace(s))
}

// Password enforces a minimum strength: at least 10 chars with some variety.
func Password(s string) bool {
	if len(s) < 10 || len(s) > 200 {
		return false
	}
	var hasLetter, hasDigit bool
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9':
			hasDigit = true
		case (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z'):
			hasLetter = true
		}
	}
	return hasLetter && hasDigit
}
