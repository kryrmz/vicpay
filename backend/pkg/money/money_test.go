package money

import "testing"

func TestNewRejectsUnknownCurrency(t *testing.T) {
	if _, err := New(100, "XXX"); err == nil {
		t.Fatal("expected error for unknown currency")
	}
	if _, err := New(100, "usd"); err != nil {
		t.Fatalf("lowercase currency should be normalized: %v", err)
	}
}

func TestAddCurrencyMismatch(t *testing.T) {
	usd := MustNew(100, "USD")
	crc := MustNew(100, "CRC")
	if _, err := usd.Add(crc); err != ErrCurrencyMismatch {
		t.Fatalf("expected ErrCurrencyMismatch, got %v", err)
	}
	sum, err := usd.Add(MustNew(50, "USD"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sum.AmountMinor() != 150 {
		t.Fatalf("expected 150, got %d", sum.AmountMinor())
	}
}

func TestString(t *testing.T) {
	cases := map[int64]string{
		1234:  "12.34 USD",
		5:     "0.05 USD",
		-9900: "-99.00 USD",
	}
	for amount, want := range cases {
		if got := MustNew(amount, "USD").String(); got != want {
			t.Errorf("amount %d: got %q want %q", amount, got, want)
		}
	}
}

func TestNegAndZero(t *testing.T) {
	m := MustNew(-500, "CRC")
	if !m.Neg().Equal(500) {
		t.Fatalf("neg failed: %s", m.Neg())
	}
	if !MustNew(0, "USD").IsZero() {
		t.Fatal("expected zero")
	}
}

// Equal is a tiny test helper.
func (m Money) Equal(amountMinor int64) bool { return m.amount == amountMinor }
