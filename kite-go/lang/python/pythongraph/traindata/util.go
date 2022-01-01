package traindata

import (
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/kiteco/kiteco/kite-golib/fileutil"
)

// LoadPackageList from the specified file path
func LoadPackageList(file string) ([]string, error) {
	f, err := fileutil.NewCachedReader(file)
	if err != nil {
		return nil, fmt.Errorf("unable to open package list %s: %v", file, err)
	}
	defer f.Close()
	contents, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("unable to read package list: %v", err)
	}

	var packages []string
	for _, line := range strings.Split(string(contents), "\n") {
		if pkg := strings.TrimSpace(line); pkg != "" {
			packages = append(packages, pkg)
		}
	}

	if len(packages) == 0 {
		return nil, fmt.Errorf("no packages found")
	}

	return packages, nil
}
