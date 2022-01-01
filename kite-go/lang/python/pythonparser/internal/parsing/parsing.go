// Package parsing implements parsing utilities common to specialized python parsers.
package parsing

// TrimMaxLines trims src so that a maximum of maxLines lines are present.
// Lines are separated by one of \n, \r, \r\n as per the Python standard.
// It returns the potentially trimmed output and a boolean indicating if the input was trimmed.
func TrimMaxLines(src []byte, maxLines uint64) (out []byte, trimmed bool) {
	if maxLines == 0 {
		return src, false
	}

	var cntNL uint64
	for i := 0; i < len(src); i++ {
		if src[i] == '\n' || src[i] == '\r' {
			cntNL++
			if cntNL >= maxLines {
				return src[:i], true
			}

			if src[i] == '\r' && (i+1) < len(src) && src[i+1] == '\n' {
				// jump over the following \n when processing \r\n
				i++
			}
		}
	}
	return src, false
}
