package auth

import "testing"

func TestHashAndVerifyPassword(t *testing.T) {
	t.Parallel()

	service := NewService("fern")
	hash, err := service.HashPassword("secret123")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	if hash == "secret123" {
		t.Fatalf("password should not be stored in plain text")
	}
	if !service.VerifyPassword(hash, "secret123") {
		t.Fatalf("expected password to verify")
	}
	if service.VerifyPassword(hash, "wrong-password") {
		t.Fatalf("expected wrong password to fail")
	}
}
