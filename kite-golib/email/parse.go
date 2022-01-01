package email

import (
	"fmt"
	"strings"

	"github.com/asaskevich/govalidator"
)

// Address represents a parsed email address
type Address struct {
	User    string
	Domain  string
	Address string `valid:"email"`
}

// ParseAddress parses an email address
func ParseAddress(email string) (Address, error) {
	if !govalidator.IsEmail(email) {
		return Address{}, fmt.Errorf("%s not a valid e-mail address", email)
	}

	addr := Address{Address: email}
	pos := strings.Index(email, "@")
	addr.User, addr.Domain = email[:pos], email[pos+1:]

	return addr, nil
}

func (a Address) canonicalUser() string {
	user := a.User

	pos := strings.Index(user, "+")
	if pos > 0 {
		user = user[:pos]
	}

	parts := strings.Split(user, ".")
	user = strings.Join(parts, "")

	return user
}

// Duplicate returns whether two email addresses referr to the same account.
func Duplicate(e1, e2 string) bool {
	if e1 == e2 {
		return true
	}

	addr1, err := ParseAddress(e1)
	if err != nil {
		return false
	}

	addr2, err := ParseAddress(e2)
	if err != nil {
		return false
	}

	if addr1.Domain != addr2.Domain {
		return false
	}

	if addr1.canonicalUser() != addr2.canonicalUser() {
		return false
	}

	return true
}
