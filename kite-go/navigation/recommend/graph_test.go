package recommend

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type graphRecommendFilesTC struct {
	currentFile   fileID
	expectedFiles []fileID
}

func TestGraphRecommendFiles(t *testing.T) {
	g := graph{
		files: map[fileID][]commitID{
			10: []commitID{1},
			20: []commitID{1, 2},
			50: []commitID{2, 3},
			30: []commitID{3, 4},
			40: []commitID{4},
		},
		editSize: map[commitID]uint32{
			1: 2,
			2: 2,
			3: 2,
			4: 2,
		},
		opts: defaultGraphOptions,
	}
	g.computeEditScores()

	tcs := []graphRecommendFilesTC{
		graphRecommendFilesTC{
			currentFile:   10,
			expectedFiles: []fileID{20, 30, 50, 40},
		},
		graphRecommendFilesTC{
			currentFile:   20,
			expectedFiles: []fileID{50, 10, 30, 40},
		},
		graphRecommendFilesTC{
			currentFile:   50,
			expectedFiles: []fileID{20, 30, 10, 40},
		},
		graphRecommendFilesTC{
			currentFile:   30,
			expectedFiles: []fileID{50, 40, 20, 10},
		},
		graphRecommendFilesTC{
			currentFile:   60,
			expectedFiles: []fileID{20, 30, 50, 10, 40},
		},
	}

	for _, tc := range tcs {
		recs := g.recommendFiles(tc.currentFile)
		var files []fileID
		for _, rec := range recs {
			files = append(files, rec.id)
		}
		assert.Equal(t, tc.expectedFiles, files)
	}
}
