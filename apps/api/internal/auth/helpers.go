package auth

import (
	"crypto/rand"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

const saltLength = 16

// hashPassword generates a random salt and hashes the password with bcrypt.
func hashPassword(password string) (salt, hash []byte, err error) {
	salt = make([]byte, saltLength)
	if _, err := rand.Read(salt); err != nil {
		return nil, nil, fmt.Errorf("generate salt: %w", err)
	}

	salted := saltPassword(salt, password)
	hash, err = bcrypt.GenerateFromPassword(salted, bcrypt.DefaultCost)
	if err != nil {
		return nil, nil, fmt.Errorf("hash password: %w", err)
	}

	return salt, hash, nil
}

// checkPassword verifies a password against the stored hash and salt.
func checkPassword(password string, salt, storedHash []byte) error {
	salted := saltPassword(salt, password)
	return bcrypt.CompareHashAndPassword(storedHash, salted)
}

// saltPassword returns salt || password without mutating the salt slice.
func saltPassword(salt []byte, password string) []byte {
	salted := make([]byte, len(salt)+len(password))
	copy(salted, salt)
	copy(salted[len(salt):], password)
	return salted
}
