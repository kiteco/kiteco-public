//go:generate go-bindata -o bindata.go -pkg notifications static/...

package notifications

import (
	"bufio"
	"bytes"
	"unsafe"

	"github.com/kiteco/kiteco/kite-golib/conversion"
)

// ProCompletionCTA gets the notification id for the Kite Pro completion CTA for the given user ID
func ProCompletionCTA(uid string) string {
	switch true {
	case userInCSV(uid, MustAsset("static/25_discount_ids.csv")):
		return "completions_cta_post_trial_25off"
	case userInCSV(uid, MustAsset("static/50_discount_ids.csv")):
		return "completions_cta_post_trial_50off"
	default:
		return conversion.CompletionsCTAPostTrial
	}
}

func userInCSV(id string, data []byte) bool {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		if scanner.Text() == id {
			return true
		}
	}
	return false
}

func bytesToString(bs []byte) string {
	return *(*string)(unsafe.Pointer(&bs))
}
