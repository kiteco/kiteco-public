package authentication

import (
	"golang.org/x/crypto/bcrypt"
)

const (
	passwordCost        = 10
	preSalt      string = "XXXXXXX"
	postSalt     string = "/$`*tgihtg45r89tgXXXXXXX"
)

// PasswordHash hashes a password.
func PasswordHash(password string) (string, error) {
	salted := preSalt + password + postSalt
	h, err := bcrypt.GenerateFromPassword([]byte(salted), passwordCost)
	if err != nil {
		return "", err
	}
	return string(h), nil

}

// PasswordMatches checkes if passwordHash and the hashed version of password match.
func PasswordMatches(passwordHash string, password string) bool {
	salted := preSalt + password + postSalt
	return bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(salted)) == nil
}
