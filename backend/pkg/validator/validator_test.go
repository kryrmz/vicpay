package validator

import "testing"

func TestEmail(t *testing.T) {
	good := []string{"a@b.co", "user.name+tag@example.com"}
	bad := []string{"", "no-at", "a@b", "a@.com", "@b.co"}
	for _, s := range good {
		if !Email(s) {
			t.Errorf("expected %q valid", s)
		}
	}
	for _, s := range bad {
		if Email(s) {
			t.Errorf("expected %q invalid", s)
		}
	}
}

func TestPhoneE164(t *testing.T) {
	if !PhoneE164("+50688881234") {
		t.Error("valid CR number rejected")
	}
	if !PhoneE164("+14155552671") {
		t.Error("valid US number rejected")
	}
	for _, s := range []string{"88881234", "+0123", "+506888812345678901", "506-8888-1234"} {
		if PhoneE164(s) {
			t.Errorf("expected %q invalid", s)
		}
	}
}

func TestPassword(t *testing.T) {
	if !Password("hunter2go1234") {
		t.Error("strong password rejected")
	}
	for _, s := range []string{"short1", "alllettersnodigits", "1234567890"} {
		if Password(s) {
			t.Errorf("weak password %q accepted", s)
		}
	}
}
