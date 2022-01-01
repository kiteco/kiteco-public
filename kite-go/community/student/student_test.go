package student

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var testDomainList = DomainLists{
	BlackList: map[string]struct{}{
		"alumni.school.edu": struct{}{},
		"notaschool.edu":    struct{}{},
	},
	WhiteList: map[string]struct{}{
		"edu":        struct{}{},
		"school.com": struct{}{},
	},
}

func TestWhiteList(t *testing.T) {
	assert.True(t, testDomainList.IsStudent("a@school.com"), "Any address from domain school.com should be allowed")
	assert.True(t, testDomainList.IsStudent("a@a_edu_domain.edu"))
	assert.True(t, testDomainList.IsStudent("a@plenty.of.subdomain.com.edu"))
}

func TestNoList(t *testing.T) {
	assert.False(t, testDomainList.IsStudent("user@gmail.com"))
	assert.False(t, testDomainList.IsStudent("wrongemail@"))
	assert.False(t, testDomainList.IsStudent("school.edu@gmail.com"))
	assert.False(t, testDomainList.IsStudent(""))
}

func TestBlackList(t *testing.T) {
	assert.False(t, testDomainList.IsStudent("user@notaschool.edu"))
	assert.False(t, testDomainList.IsStudent("contact@test.alumni.school.edu"))
}
