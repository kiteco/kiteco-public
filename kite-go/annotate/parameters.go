package annotate

import "strconv"

var (
	// The standard parameters substituted for in all code examples
	globalSubstitutions = map[string]string{
		"IMAGE_WIDTH_PIXELS":  "280",
		"IMAGE_HEIGHT_PIXELS": "210",
		"IMAGE_WIDTH_INCHES":  "4",
		"IMAGE_HEIGHT_INCHES": "3",
		"IMAGE_DPI":           "70",
		"HTTP_PORT":           strconv.Itoa(httpInternalPort),
	}
)
