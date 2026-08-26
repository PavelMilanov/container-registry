package auth

import (
	"errors"
	"strings"
	"testing"
)

func newTestPasswordHasher() *PasswordHasher {
	return &PasswordHasher{
		memory:      8 * 1024,
		iterations:  1,
		parallelism: 1,
		saltLength:  16,
		keyLength:   32,
	}
}

func TestPasswordHasherHashAndVerify(t *testing.T) {
	hasher := newTestPasswordHasher()
	first, err := hasher.Hash("correct horse battery staple")
	if err != nil {
		t.Fatal(err)
	}
	second, err := hasher.Hash("correct horse battery staple")
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("equal passwords received equal salted hashes")
	}

	valid, needsUpgrade, err := hasher.Verify(
		first,
		"correct horse battery staple",
	)
	if err != nil {
		t.Fatal(err)
	}
	if !valid || needsUpgrade {
		t.Fatalf("valid = %t, needsUpgrade = %t", valid, needsUpgrade)
	}

	valid, _, err = hasher.Verify(first, "wrong password")
	if err != nil {
		t.Fatal(err)
	}
	if valid {
		t.Fatal("wrong password was accepted")
	}
}

func TestPasswordHasherRejectsInvalidInput(t *testing.T) {
	hasher := newTestPasswordHasher()

	if _, err := hasher.Hash(""); !errors.Is(err, ErrEmptyPassword) {
		t.Fatalf("empty password error = %v", err)
	}
	if _, err := hasher.Hash(strings.Repeat("a", maxPasswordBytes+1)); !errors.Is(err, ErrPasswordTooLong) {
		t.Fatalf("long password error = %v", err)
	}
	if _, _, err := hasher.Verify("not-a-hash", "password"); !errors.Is(err, ErrInvalidPasswordHash) {
		t.Fatalf("invalid hash error = %v", err)
	}
}

func TestPasswordHasherRejectsUnsafeArgonParameters(t *testing.T) {
	hasher := newTestPasswordHasher()
	hash, err := hasher.Hash("password")
	if err != nil {
		t.Fatal(err)
	}
	unsafeHash := strings.Replace(
		hash,
		"m=8192,t=1,p=1",
		"m=1048577,t=1,p=1",
		1,
	)

	if _, _, err := hasher.Verify(unsafeHash, "password"); !errors.Is(err, ErrInvalidPasswordHash) {
		t.Fatalf("unsafe hash error = %v", err)
	}
}

func TestPasswordHasherMarksDifferentParametersForUpgrade(t *testing.T) {
	current := newTestPasswordHasher()
	previous := newTestPasswordHasher()
	previous.iterations = 2
	hash, err := previous.Hash("password")
	if err != nil {
		t.Fatal(err)
	}

	valid, needsUpgrade, err := current.Verify(hash, "password")
	if err != nil {
		t.Fatal(err)
	}
	if !valid || !needsUpgrade {
		t.Fatalf("valid = %t, needsUpgrade = %t", valid, needsUpgrade)
	}
}
