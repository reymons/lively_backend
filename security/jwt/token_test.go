package jwt

import (
	"math/rand"
	"testing"
	"time"
)

func TestToken_CreateAndVerify(t *testing.T) {
	user := User{ID: rand.Uint64()}
	secret := "qohUiI5orYqudTMLC1EKmQ9qVBl0OPDfOqp16IWW6kR"
	token := NewToken(secret, time.Minute*5)

	tkn, err := token.Create(&user)
	if err != nil {
		t.Fatalf("create token: %v", err)
	}

	var got User
	if err := token.Verify(tkn, &got); err != nil {
		t.Fatalf("verify token: %v", err)
	}

	if user.ID != got.ID {
		t.Errorf("invalid user ID: expected %d, got %d", user.ID, got.ID)
	}
}
