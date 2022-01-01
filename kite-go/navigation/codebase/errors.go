package codebase

// ErrorString converts a codebase error into a string for metrics
func ErrorString(err error) string {
	switch err {
	case ErrPathHasUnsupportedExtension:
		return "unsupported file extension"
	case ErrPathNotInSupportedProject:
		return "not in git project"
	case ErrProjectStillIndexing:
		return "still indexing"
	default:
		return ""
	}
}
