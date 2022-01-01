package git

import (
	"encoding/json"
	"log"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	alphaBeta        = repoKey("/alpha/beta")
	alphaBetaPartial = repoCache{
		Commits: map[CommitHash][]int{
			"123":  []int{0, 3, 4},
			"5432": []int{2, 1, 4, 3},
		},
		Files: []File{"one", "four", "five", "two", "three"},
	}
	alphaBetaFull = repoCache{
		Commits: alphaBetaPartial.Commits,
		Files:   alphaBetaPartial.Files,
		inverted: map[File]int{
			"one":   0,
			"four":  1,
			"five":  2,
			"two":   3,
			"three": 4,
		},
	}

	gammaDelta        = repoKey("/gamma/delta")
	gammaDeltaPartial = repoCache{
		Commits: map[CommitHash][]int{
			"37": []int{2, 0},
			"9":  []int{1},
		},
		Files: []File{"seven", "nine", "three"},
	}
	gammaDeltaFull = repoCache{
		Commits: gammaDeltaPartial.Commits,
		Files:   gammaDeltaPartial.Files,
		inverted: map[File]int{
			"seven": 0,
			"nine":  1,
			"three": 2,
		},
	}
)

func TestRepoBundleAdd(t *testing.T) {
	r := newRepoBundle()
	r.add(alphaBeta, alphaBetaFull)
	r.add(gammaDelta, gammaDeltaFull)
	r.add(alphaBeta, alphaBetaFull)

	expected := repoBundle{
		Repos: map[repoKey]repoCache{
			alphaBeta:  alphaBetaFull,
			gammaDelta: gammaDeltaFull,
		},
		EvictionOrder: []repoKey{gammaDelta, alphaBeta},
	}
	require.Equal(t, expected, r)
}

func TestRepoBundleGet(t *testing.T) {
	r := repoBundle{
		Repos: map[repoKey]repoCache{
			alphaBeta:  alphaBetaPartial,
			gammaDelta: gammaDeltaPartial,
		},
		EvictionOrder: []repoKey{alphaBeta, gammaDelta},
	}
	alphaBetaGot, alphaBetaHit := r.get(alphaBeta)
	gammaDeltaGot, gammaDeltaHit := r.get(gammaDelta)
	epsilonGot, epsilonHit := r.get(repoKey("epsilon"))
	assert.Equal(t, alphaBetaFull, alphaBetaGot)
	assert.Equal(t, true, alphaBetaHit)
	assert.Equal(t, gammaDeltaFull, gammaDeltaGot)
	assert.Equal(t, true, gammaDeltaHit)
	assert.Equal(t, newRepoCache(), epsilonGot)
	assert.Equal(t, false, epsilonHit)
}

type marshalTC struct {
	maxSize       int64
	expected      repoBundle
	expectedEvict bool
}

func TestMarshal(t *testing.T) {
	tcs := []marshalTC{
		marshalTC{
			maxSize: 300,
			expected: repoBundle{
				Repos: map[repoKey]repoCache{
					alphaBeta:  alphaBetaPartial,
					gammaDelta: gammaDeltaPartial,
				},
				EvictionOrder: []repoKey{alphaBeta, gammaDelta},
			},
			expectedEvict: false,
		},
		marshalTC{
			maxSize: 200,
			expected: repoBundle{
				Repos: map[repoKey]repoCache{
					gammaDelta: gammaDeltaPartial,
				},
				EvictionOrder: []repoKey{gammaDelta},
			},
			expectedEvict: true,
		},
		marshalTC{
			maxSize: 100,
			expected: repoBundle{
				Repos:         map[repoKey]repoCache{},
				EvictionOrder: []repoKey{},
			},
			expectedEvict: true,
		},
	}

	for _, tc := range tcs {
		original := repoBundle{
			Repos: map[repoKey]repoCache{
				alphaBeta:  alphaBetaPartial,
				gammaDelta: gammaDeltaFull,
			},
			EvictionOrder: []repoKey{alphaBeta, gammaDelta},
		}
		data, evict, err := original.evictAndMarshal(tc.maxSize)
		require.NoError(t, err)
		require.Equal(t, tc.expectedEvict, evict)

		reconstructed, err := unmarshalRepoBundle(data)
		require.NoError(t, err)
		assert.Equal(t, tc.expected, reconstructed)

		// data format should be valid json
		var msg json.RawMessage
		err = json.Unmarshal(data, &msg)
		require.NoError(t, err)
	}
}

func TestRepoCacheGet(t *testing.T) {
	commit, ok, err := alphaBetaFull.get("123")
	require.NoError(t, err)
	assert.True(t, ok)

	expected := Commit{
		Hash:  "123",
		Files: []File{"one", "two", "three"},
	}
	assert.Equal(t, expected, commit)
}

func TestRepoCacheAdd(t *testing.T) {
	repo := newRepoCache()
	alpha := Commit{
		Hash:  "123",
		Files: []File{"one", "two", "three"},
	}
	beta := Commit{
		Hash:  "5432",
		Files: []File{"five", "four", "three", "two"},
	}
	repo.add("123", alpha)
	repo.add("5432", beta)

	alphaRecovered, ok, err := repo.get("123")
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, alpha, alphaRecovered)

	betaRecovered, ok, err := repo.get("5432")
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, beta, betaRecovered)
}

func BenchmarkStressEvictAndMarshal(b *testing.B) {
	big := newRepoCache()
	for i := 0; i < 1e4; i++ {
		hash := CommitHash(strconv.Itoa(i))
		commit := Commit{
			Files: []File{"alpha", "beta", "gamma", "delta"},
		}
		big.add(hash, commit)
	}

	var evictAndMarshalDuration time.Duration
	for i := 0; i < b.N; i++ {
		bundle := newRepoBundle()
		for i := 0; i < 100; i++ {
			bundle.add(repoKey(strconv.Itoa(i)), big)
		}
		start := time.Now()
		_, _, err := bundle.evictAndMarshal(1e7)
		if err != nil {
			log.Fatal(err)
		}
		evictAndMarshalDuration += time.Since(start)
	}
	b.ReportMetric(float64(evictAndMarshalDuration.Seconds())/float64(b.N), "s/evictAndMarshal")
	b.ReportMetric(0, "ns/op")
}
