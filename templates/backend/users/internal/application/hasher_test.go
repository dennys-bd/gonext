package application_test

import (
	"testing"

	"[PROJECT-NAME]/backend/users/internal/application"
)

func TestArgon2Hasher_HashAndVerify(t *testing.T) {
	h := application.NewArgon2Hasher()

	hash, err := h.Hash("correct horse battery staple")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if hash == "correct horse battery staple" {
		t.Fatal("expected the hash not to be the plaintext")
	}

	ok, err := h.Verify("correct horse battery staple", hash)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if !ok {
		t.Fatal("expected the correct password to verify")
	}

	ok, err = h.Verify("wrong password", hash)
	if err != nil {
		t.Fatalf("verify wrong password: %v", err)
	}
	if ok {
		t.Fatal("expected a wrong password not to verify")
	}
}

func TestArgon2Hasher_HashIsSalted(t *testing.T) {
	h := application.NewArgon2Hasher()

	first, err := h.Hash("same password")
	if err != nil {
		t.Fatalf("first hash: %v", err)
	}
	second, err := h.Hash("same password")
	if err != nil {
		t.Fatalf("second hash: %v", err)
	}

	if first == second {
		t.Fatal("expected two hashes of the same password to differ (random salt)")
	}
}

func TestArgon2Hasher_Verify_MalformedHash(t *testing.T) {
	h := application.NewArgon2Hasher()

	tests := []struct {
		name string
		hash string
	}{
		{"empty", ""},
		{"not a phc string", "plaintext"},
		{"wrong algorithm", "$argon2i$v=19$m=65536,t=1,p=4$c2FsdA$ZGlnZXN0"},
		{"unparsable parameters", "$argon2id$v=19$m=abc,t=1,p=4$c2FsdA$ZGlnZXN0"},
		{"undecodable salt", "$argon2id$v=19$m=65536,t=1,p=4$!!!$ZGlnZXN0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok, err := h.Verify("whatever", tt.hash)
			if err == nil {
				t.Fatal("expected an error for a malformed hash")
			}
			if ok {
				t.Fatal("expected a malformed hash not to verify")
			}
		})
	}
}
