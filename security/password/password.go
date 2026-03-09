package password

import "golang.org/x/crypto/bcrypt"

type Manager interface {
	HashPassword(password string) (string, error)

	VerifyPassword(password string, hash string) bool
}

type manager struct{}

func NewManager() Manager {
	return &manager{}
}

func (m *manager) HashPassword(password string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(hashed), err
}

func (m *manager) VerifyPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
