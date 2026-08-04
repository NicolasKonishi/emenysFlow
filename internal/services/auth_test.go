package services

import "testing"

func TestPasswordHashAndVerification(t *testing.T) {
	hash, err := HashPassword("admin123")
	if err != nil {
		t.Fatal(err)
	}
	if hash == "admin123" {
		t.Fatal("password was not hashed")
	}
	if !VerifyPassword(hash, "admin123") {
		t.Fatal("valid password rejected")
	}
	if VerifyPassword(hash, "wrong-password") {
		t.Fatal("invalid password accepted")
	}
}
