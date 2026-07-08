package hash

import "testing"

// fastParams keep the tests quick; production uses DefaultParams.
var fastParams = Params{Memory: 8 * 1024, Iterations: 1, Parallelism: 1, SaltLength: 16, KeyLength: 32}

func TestHashVerifyRoundTrip(t *testing.T) {
	encoded, err := HashWithParams("correct horse battery staple", fastParams)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	ok, err := Verify("correct horse battery staple", encoded)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if !ok {
		t.Fatal("expected password to verify")
	}
	bad, err := Verify("wrong password", encoded)
	if err != nil {
		t.Fatalf("verify wrong: %v", err)
	}
	if bad {
		t.Fatal("wrong password must not verify")
	}
}

func TestVerifyRejectsMalformed(t *testing.T) {
	if _, err := Verify("x", "not-a-hash"); err != ErrInvalidHash {
		t.Fatalf("expected ErrInvalidHash, got %v", err)
	}
}

func TestHashIsSalted(t *testing.T) {
	a, _ := HashWithParams("same", fastParams)
	b, _ := HashWithParams("same", fastParams)
	if a == b {
		t.Fatal("two hashes of the same password must differ (random salt)")
	}
}
