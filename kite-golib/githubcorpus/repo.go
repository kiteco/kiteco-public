package githubcorpus

import (
	"strconv"
	"strings"

	"github.com/kiteco/kiteco/kite-golib/errors"
)

const nameSepRepoCorpus = "__"

// RepoCorpusFilename for the provided owner/name and id
func RepoCorpusFilename(owner, name string, id int) string {
	base := strings.Join([]string{
		"github.com",
		strconv.Itoa(id),
		owner, name,
	}, nameSepRepoCorpus)
	return base + ".tar.gz"
}

// ParseRepoCorpusFilename from the provided filename,
// returns owner,name,id,err
func ParseRepoCorpusFilename(fn string) (string, string, int, error) {
	fnTrimmed := strings.TrimSuffix(fn, ".tar.gz")
	parts := strings.Split(fnTrimmed, nameSepRepoCorpus)
	if len(parts) != 4 {
		return "", "", 0, errors.New("invalid repo corpus filename %s", fn)
	}
	ids, owner, name := parts[1], parts[2], parts[3]

	id, err := strconv.Atoi(ids)
	if err != nil {
		return "", "", 0, errors.New("invalid repo corpus filename %s: %v", fn, err)
	}
	return owner, name, id, nil
}
