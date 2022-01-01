package metrics

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type logger interface {
	Log()
}

type logTC struct {
	events   []logger
	expected aggregated
}

func TestLog(t *testing.T) {
	tcs := []logTC{
		logTC{
			events: []logger{
				Index{
					Duration: 2 * time.Second,
					NumFiles: 7,
				},
			},
			expected: aggregated{
				index: Index{
					Duration: 7 * time.Second,
					NumFiles: 10,
				},
				indexCount: 3,
				rank: Rank{
					Duration:          11 * time.Second,
					NumFiles:          23,
					NumRefreshedFiles: 8,
				},
				rankCount: 7,
				batch: Batch{
					Duration: 54 * time.Second,
					NumFiles: 481,
				},
				batchCount: 21,
				gitCache: GitCache{
					NumRepos:   3,
					NumCommits: 453,
					NumFiles:   314,
					NumBytes:   56331,
					Hits:       2,
					Evictions:  4,
				},
				gitCacheCount: 5,
			},
		},
		logTC{
			events: []logger{
				Index{
					Duration: 2 * time.Second,
					NumFiles: 7,
				},
				Index{
					Duration: 7 * time.Second,
					NumFiles: 5,
				},
			},
			expected: aggregated{
				index: Index{
					Duration: 14 * time.Second,
					NumFiles: 15,
				},
				indexCount: 4,
				rank: Rank{
					Duration:          11 * time.Second,
					NumFiles:          23,
					NumRefreshedFiles: 8,
				},
				rankCount: 7,
				batch: Batch{
					Duration: 54 * time.Second,
					NumFiles: 481,
				},
				batchCount: 21,
				gitCache: GitCache{
					NumRepos:   3,
					NumCommits: 453,
					NumFiles:   314,
					NumBytes:   56331,
					Hits:       2,
					Evictions:  4,
				},
				gitCacheCount: 5,
			},
		},
		logTC{
			events: []logger{
				Rank{
					Duration: 2 * time.Second,
					NumFiles: 7,
				},
				Index{
					Duration: 7 * time.Second,
					NumFiles: 5,
				},
			},
			expected: aggregated{
				index: Index{
					Duration: 12 * time.Second,
					NumFiles: 8,
				},
				indexCount: 3,
				rank: Rank{
					Duration:          13 * time.Second,
					NumFiles:          30,
					NumRefreshedFiles: 8,
				},
				rankCount: 8,
				batch: Batch{
					Duration: 54 * time.Second,
					NumFiles: 481,
				},
				batchCount: 21,
				gitCache: GitCache{
					NumRepos:   3,
					NumCommits: 453,
					NumFiles:   314,
					NumBytes:   56331,
					Hits:       2,
					Evictions:  4,
				},
				gitCacheCount: 5,
			},
		},
		logTC{
			events: []logger{
				GitCache{
					NumRepos:   2,
					NumCommits: 100,
					NumFiles:   200,
					NumBytes:   1000,
					Hits:       3,
					Evictions:  7,
				},
				Rank{
					Duration: 2 * time.Second,
					NumFiles: 7,
				},
				Batch{
					Duration: 7 * time.Second,
					NumFiles: 5,
				},
				GitCache{
					NumRepos:   1,
					NumCommits: 200,
					NumFiles:   300,
					NumBytes:   2000,
					Hits:       4,
					Evictions:  10,
				},
			},
			expected: aggregated{
				index: Index{
					Duration: 5 * time.Second,
					NumFiles: 3,
				},
				indexCount: 2,
				rank: Rank{
					Duration:          13 * time.Second,
					NumFiles:          30,
					NumRefreshedFiles: 8,
				},
				rankCount: 8,
				batch: Batch{
					Duration: 61 * time.Second,
					NumFiles: 486,
				},
				batchCount: 22,
				gitCache: GitCache{
					NumRepos:   6,
					NumCommits: 753,
					NumFiles:   814,
					NumBytes:   59331,
					Hits:       9,
					Evictions:  21,
				},
				gitCacheCount: 7,
			},
		},
	}

	before := aggregated{
		index: Index{
			Duration: 5 * time.Second,
			NumFiles: 3,
		},
		indexCount: 2,
		rank: Rank{
			Duration:          11 * time.Second,
			NumFiles:          23,
			NumRefreshedFiles: 8,
		},
		rankCount: 7,
		batch: Batch{
			Duration: 54 * time.Second,
			NumFiles: 481,
		},
		batchCount: 21,
		gitCache: GitCache{
			NumRepos:   3,
			NumCommits: 453,
			NumFiles:   314,
			NumBytes:   56331,
			Hits:       2,
			Evictions:  4,
		},
		gitCacheCount: 5,
	}

	for _, tc := range tcs {
		agg = before
		for _, event := range tc.events {
			event.Log()
		}
		require.Equal(t, tc.expected, agg)
	}
}

func TestRead(t *testing.T) {
	before := aggregated{
		index: Index{
			Duration: 5 * time.Second,
			NumFiles: 3,
		},
		indexCount: 2,
		rank: Rank{
			Duration:          11 * time.Second,
			NumFiles:          23,
			NumRefreshedFiles: 8,
		},
		rankCount: 7,
		batch: Batch{
			Duration: 54 * time.Second,
			NumFiles: 481,
		},
		batchCount: 21,
		gitCache: GitCache{
			NumRepos:   3,
			NumCommits: 453,
			NumFiles:   314,
			NumBytes:   56331,
			Hits:       2,
			Evictions:  4,
		},
		gitCacheCount: 5,
	}
	expected := map[string]int64{
		"nav_index_duration_ms":        5000,
		"nav_index_num_files":          3,
		"nav_index_count":              2,
		"nav_rank_duration_ms":         11000,
		"nav_rank_num_files":           23,
		"nav_rank_num_refreshed_files": 8,
		"nav_rank_count":               7,
		"nav_batch_duration_ms":        54000,
		"nav_batch_num_files":          481,
		"nav_batch_count":              21,
		"nav_git_cache_num_repos":      3,
		"nav_git_cache_num_commits":    453,
		"nav_git_cache_num_files":      314,
		"nav_git_cache_num_bytes":      56331,
		"nav_git_cache_hits":           2,
		"nav_git_cache_evictions":      4,
		"nav_git_cache_count":          5,
	}
	agg = before
	withoutClear := Read(false)
	require.Equal(t, expected, withoutClear)
	require.Equal(t, before, agg)

	withClear := Read(true)
	var cleared aggregated
	require.Equal(t, expected, withClear)
	require.Equal(t, cleared, agg)
}
