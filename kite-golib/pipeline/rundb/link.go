package rundb

import (
	"fmt"
	"strings"
)

// RenderLink renders the provided url as an html link
func RenderLink(display, url string) string {
	if display == "" {
		display = url
	}
	return fmt.Sprintf(`<a href="%s" target="_blank">%s</a>`, url, display)
}

// RenderS3ObjectLink renders the provided key to an s3 bucket as a link
func RenderS3ObjectLink(display, key string) string {
	key = strings.TrimPrefix(key, "/")
	key = strings.TrimPrefix(key, "s3://")
	return RenderLink(display, fmt.Sprintf("https://s3.console.aws.amazon.com/s3/object/%s?region=us-west-1&tab=overview", key))
}

// RenderS3DirLink renders the provided ket to an s3 directory as a link
func RenderS3DirLink(display, key string) string {
	key = strings.TrimPrefix(key, "/")
	key = strings.TrimPrefix(key, "s3://")
	if !strings.HasSuffix(key, "/") {
		key += "/"
	}
	return RenderLink(display, fmt.Sprintf("https://s3.console.aws.amazon.com/s3/buckets/%s?region=us-west-1&tab=overview", key))
}
