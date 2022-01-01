package localtraining

import (
	"log"
	"math/rand"
	"os"
	"sort"

	"github.com/kiteco/kiteco/kite-golib/errors"
)

// SplitType determines how local files are split into train/validate/test sets
type SplitType string

const (
	// RandomSplit ...
	RandomSplit SplitType = "random"
	// LastModifiedTimeSplit ...
	LastModifiedTimeSplit SplitType = "last_modified_time"
)

// Current memory limit for 16gb machines
const maxFiles = 2000

// Split local files into train, validate and test set
func Split(localFiles []string, seed int64, splitType SplitType,
	trainRatio, validateRatio, testRatio float64) ([]string, []string, []string, error) {

	if trainRatio+validateRatio+testRatio != 1 {
		return nil, nil, nil, errors.New("invalid data splitting ratios")
	}

	switch splitType {
	case RandomSplit:
		r := rand.New(rand.NewSource(seed))
		r.Shuffle(len(localFiles), func(i, j int) {
			localFiles[i], localFiles[j] = localFiles[j], localFiles[i]
		})
		if len(localFiles) > maxFiles {
			log.Println("truncating to 2k files for training")
			localFiles = localFiles[:maxFiles]
		}
	case LastModifiedTimeSplit:
		sort.Slice(localFiles, func(i, j int) bool {
			infoI, errI := os.Stat(localFiles[i])
			infoJ, errJ := os.Stat(localFiles[j])
			return errI == nil && errJ == nil && infoI.ModTime().Before(infoJ.ModTime())
		})
	default:
		return nil, nil, nil, errors.New("unsupported splitting method '%s', must be random_file|last_modified_time", splitType)
	}

	firstCut := int(float64(len(localFiles)) * trainRatio)
	secondCut := int(float64(len(localFiles)) * (trainRatio + validateRatio))
	trainFiles := localFiles[:firstCut]
	validateFiles := localFiles[firstCut:secondCut]
	testFiles := localFiles[secondCut:]
	return trainFiles, validateFiles, testFiles, nil
}
