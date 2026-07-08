// Package money represents monetary amounts as integer minor units to avoid
// floating-point rounding errors. A Money value is an (amount, currency) pair
// where amount is expressed in the smallest indivisible unit of the currency
// (e.g. cents for USD, centimos for CRC).
package money

import (
	"errors"
	"fmt"
	"strings"
)

// Currency is an ISO 4217 alphabetic code.
type Currency string

// Supported currencies and the number of minor units per major unit.
var minorUnits = map[Currency]int{
	"USD": 100,
	"CRC": 100,
	"EUR": 100,
	"PAB": 100,
	"GTQ": 100,
}

// ErrCurrencyMismatch is returned when an operation mixes two currencies.
var ErrCurrencyMismatch = errors.New("money: currency mismatch")

// ErrUnknownCurrency is returned for a currency not in the supported set.
var ErrUnknownCurrency = errors.New("money: unknown currency")

// Money is an immutable amount in minor units tied to a currency.
type Money struct {
	amount   int64
	currency Currency
}

// New builds a Money value. It validates the currency is supported.
func New(amountMinor int64, currency Currency) (Money, error) {
	c := Currency(strings.ToUpper(string(currency)))
	if _, ok := minorUnits[c]; !ok {
		return Money{}, fmt.Errorf("%w: %q", ErrUnknownCurrency, currency)
	}
	return Money{amount: amountMinor, currency: c}, nil
}

// MustNew is like New but panics on error. Intended for tests and constants.
func MustNew(amountMinor int64, currency Currency) Money {
	m, err := New(amountMinor, currency)
	if err != nil {
		panic(err)
	}
	return m
}

// AmountMinor returns the raw amount in minor units.
func (m Money) AmountMinor() int64 { return m.amount }

// Currency returns the ISO code.
func (m Money) Currency() Currency { return m.currency }

// IsZero reports whether the amount is zero.
func (m Money) IsZero() bool { return m.amount == 0 }

// Add returns m+other, erroring on a currency mismatch.
func (m Money) Add(other Money) (Money, error) {
	if m.currency != other.currency {
		return Money{}, ErrCurrencyMismatch
	}
	return Money{amount: m.amount + other.amount, currency: m.currency}, nil
}

// Neg returns the additive inverse.
func (m Money) Neg() Money { return Money{amount: -m.amount, currency: m.currency} }

// IsSupported reports whether a currency code is known.
func IsSupported(currency Currency) bool {
	_, ok := minorUnits[Currency(strings.ToUpper(string(currency)))]
	return ok
}

// String renders the amount in major units with the currency code, e.g. "12.34 USD".
func (m Money) String() string {
	div := int64(minorUnits[m.currency])
	sign := ""
	a := m.amount
	if a < 0 {
		sign = "-"
		a = -a
	}
	return fmt.Sprintf("%s%d.%02d %s", sign, a/div, a%div, m.currency)
}
