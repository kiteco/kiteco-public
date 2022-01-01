package pythonlocal

import (
	"fmt"
	"strings"

	"github.com/kiteco/kiteco/kite-golib/errors"
)

// TODO(juan): this should be removed
// once we sort out how we want to handle paths
// from different OSes in a unified manner.
func fromUnix(unix string) (string, error) {
	if !strings.HasPrefix(unix, "/windows/") {
		return unix, nil
	}
	sub := strings.TrimPrefix(unix, "/windows/")
	pos := strings.Index(sub, "/")
	if pos == -1 {
		return "", errors.Errorf("path is missing volume: %s", unix)
	}

	vol := sub[:pos]
	winpath := strings.Replace(sub[pos:], `/`, `\`, -1)

	if vol == "unc" {
		// will result in a path like \\machine\share\path\to\src.py
		return `\` + winpath, nil
	}
	if len(vol) == 0 {
		return "", errors.Errorf("no volume")
	}
	// will result in a path like c:\path\to\src.py
	return fmt.Sprintf(`%s:%s`, vol, winpath), nil
}
