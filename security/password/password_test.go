package password

import (
	"math/rand"
	"testing"
)

func TestVerifyPassword(t *testing.T) {
	t.Parallel()
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		t.Fatalf("generate password: %v", err)
	}
	password := string(buf)
	manager := NewManager()
	hashedPassword, err := manager.HashPassword(password)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	if !manager.VerifyPassword(password, hashedPassword) {
		t.Error("VerifyPassword() = false; want true")
	}
}
