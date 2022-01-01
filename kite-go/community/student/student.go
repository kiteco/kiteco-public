package student

import (
	"compress/gzip"
	"encoding/json"
	"io/ioutil"
	"strings"

	"github.com/kiteco/kiteco/kite-golib/fileutil"
)

// PathDomainList is the path to the s3 file containing the list of whitelisted and blacklisted domains
// for student email address checking
const PathDomainList = "s3://kite-data/swot-student-domains/domain-lists_2020-10-27T13:47:41.json.gz"

// DomainLists contains a whitelist and a blacklist that allows to test if a email address is part of an
// academic domain
type DomainLists struct {
	BlackList map[string]struct{} `json:"black_list"`
	WhiteList map[string]struct{} `json:"white_list"`
}

func checkSet(domain string, dots []int, set map[string]struct{}) bool {
	for _, i := range dots {
		d := domain[i:]
		if _, ok := set[d]; ok {
			return true
		}
	}
	return false
}

// IsStudent tests if an email address is part of an academic domain (present in whitelist and not in blacklist)
func (d DomainLists) IsStudent(email string) bool {
	domain, dots := splitEmail(email)
	return !checkSet(domain, dots, d.BlackList) && checkSet(domain, dots, d.WhiteList)
}

func splitEmail(email string) (string, []int) {
	// Inspired from swot code
	// 	email.trim().toLowerCase().substringAfter('@').substringAfter("://").substringBefore(':').split('.').reversed()
	// Skipping the reverse, it's done in checkSet
	email = strings.ToLower(strings.TrimSpace(email))
	if i := strings.Index(email, "@"); i != -1 {
		email = email[i+1:]
	}
	if i := strings.Index(email, "://"); i != -1 {
		email = email[i+1:]
	}
	if i := strings.Index(email, ":"); i != -1 {
		email = email[:i]
	}
	var dots []int
	for i := strings.LastIndex(email, "."); i >= 0; i = strings.LastIndex(email[:i], ".") {
		// We store the start position of the possible domain (every block starting after a dot
		dots = append(dots, i+1)
	}
	// We also need to compare the full domain of email address, so we add 0 (email starts after the @ at this point
	dots = append(dots, 0)
	return email, dots
}

// LoadStudentDomains loads the list of domains from s3 to use for IsStudent test
// It's using the const PathDomainList to select which file to use
func LoadStudentDomains() (*DomainLists, error) {
	zReader, err := fileutil.NewCachedReader(PathDomainList)
	if err != nil {
		return nil, err
	}
	defer zReader.Close()
	gr, err := gzip.NewReader(zReader)
	if err != nil {
		return nil, err
	}
	defer gr.Close()
	data, err := ioutil.ReadAll(gr)
	if err != nil {
		return nil, err
	}
	var result DomainLists
	if err = json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
