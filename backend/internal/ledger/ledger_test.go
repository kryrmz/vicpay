package ledger

import "testing"

func TestValidate(t *testing.T) {
	usd := "USD"
	tests := []struct {
		name string
		in   PostingInput
		want error
	}{
		{
			name: "balanced transfer",
			in: PostingInput{Entries: []EntryInput{
				{Account: System("SYSTEM:EXTERNAL:USD"), Direction: Debit, AmountMinor: 1000, Currency: usd},
				{Account: Wallet("u1"), Direction: Credit, AmountMinor: 1000, Currency: usd},
			}},
			want: nil,
		},
		{
			name: "too few entries",
			in: PostingInput{Entries: []EntryInput{
				{Account: Wallet("u1"), Direction: Credit, AmountMinor: 1000, Currency: usd},
			}},
			want: ErrTooFewEntries,
		},
		{
			name: "unbalanced",
			in: PostingInput{Entries: []EntryInput{
				{Account: System("SYSTEM:EXTERNAL:USD"), Direction: Debit, AmountMinor: 1000, Currency: usd},
				{Account: Wallet("u1"), Direction: Credit, AmountMinor: 999, Currency: usd},
			}},
			want: ErrUnbalanced,
		},
		{
			name: "non-positive amount",
			in: PostingInput{Entries: []EntryInput{
				{Account: System("SYSTEM:EXTERNAL:USD"), Direction: Debit, AmountMinor: 0, Currency: usd},
				{Account: Wallet("u1"), Direction: Credit, AmountMinor: 0, Currency: usd},
			}},
			want: ErrBadAmount,
		},
		{
			name: "multi-currency balanced (FX)",
			in: PostingInput{Entries: []EntryInput{
				{Account: Wallet("u1"), Direction: Debit, AmountMinor: 1000, Currency: "USD"},
				{Account: System("SYSTEM:SUSPENSE:USD"), Direction: Credit, AmountMinor: 1000, Currency: "USD"},
				{Account: System("SYSTEM:SUSPENSE:CRC"), Direction: Debit, AmountMinor: 520000, Currency: "CRC"},
				{Account: Wallet("u1"), Direction: Credit, AmountMinor: 520000, Currency: "CRC"},
			}},
			want: nil,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := Validate(tc.in); got != tc.want {
				t.Fatalf("Validate() = %v, want %v", got, tc.want)
			}
		})
	}
}
