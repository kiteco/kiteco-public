package metrics

import (
	"sync"
	"time"
)

var (
	agg aggregated
	m   sync.Mutex
)

// Read ...
func Read(clear bool) map[string]int64 {
	m.Lock()
	defer m.Unlock()

	values := map[string]int64{
		"nav_index_duration_ms":        agg.index.Duration.Milliseconds(),
		"nav_index_num_files":          agg.index.NumFiles,
		"nav_index_count":              agg.indexCount,
		"nav_rank_duration_ms":         agg.rank.Duration.Milliseconds(),
		"nav_rank_num_files":           agg.rank.NumFiles,
		"nav_rank_num_refreshed_files": agg.rank.NumRefreshedFiles,
		"nav_rank_count":               agg.rankCount,
		"nav_batch_duration_ms":        agg.batch.Duration.Milliseconds(),
		"nav_batch_num_files":          agg.batch.NumFiles,
		"nav_batch_count":              agg.batchCount,
		"nav_git_cache_num_repos":      agg.gitCache.NumRepos,
		"nav_git_cache_num_commits":    agg.gitCache.NumCommits,
		"nav_git_cache_num_files":      agg.gitCache.NumFiles,
		"nav_git_cache_num_bytes":      agg.gitCache.NumBytes,
		"nav_git_cache_hits":           agg.gitCache.Hits,
		"nav_git_cache_evictions":      agg.gitCache.Evictions,
		"nav_git_cache_count":          agg.gitCacheCount,
	}
	if clear {
		agg = aggregated{}
	}
	return values
}

type aggregated struct {
	index         Index
	indexCount    int64
	rank          Rank
	rankCount     int64
	batch         Batch
	batchCount    int64
	gitCache      GitCache
	gitCacheCount int64
}

// Index ...
type Index struct {
	Duration time.Duration
	NumFiles int64
}

// Rank ...
type Rank struct {
	Duration          time.Duration
	NumFiles          int64
	NumRefreshedFiles int64
}

// Batch ...
type Batch struct {
	Duration time.Duration
	NumFiles int64
}

// GitCache ...
type GitCache struct {
	NumRepos   int64
	NumCommits int64
	NumFiles   int64
	NumBytes   int64
	Hits       int64
	Evictions  int64
}

// Log ...
func (i Index) Log() {
	m.Lock()
	defer m.Unlock()

	agg.index.Duration += i.Duration
	agg.index.NumFiles += i.NumFiles
	agg.indexCount++
}

// Log ...
func (r Rank) Log() {
	m.Lock()
	defer m.Unlock()

	agg.rank.Duration += r.Duration
	agg.rank.NumFiles += r.NumFiles
	agg.rank.NumRefreshedFiles += r.NumRefreshedFiles
	agg.rankCount++
}

// Log ...
func (b Batch) Log() {
	m.Lock()
	defer m.Unlock()

	agg.batch.Duration += b.Duration
	agg.batch.NumFiles += b.NumFiles
	agg.batchCount++
}

// Log ...
func (g GitCache) Log() {
	m.Lock()
	defer m.Unlock()

	agg.gitCache.NumRepos += g.NumRepos
	agg.gitCache.NumCommits += g.NumCommits
	agg.gitCache.NumFiles += g.NumFiles
	agg.gitCache.NumBytes += g.NumBytes
	agg.gitCache.Hits += g.Hits
	agg.gitCache.Evictions += g.Evictions
	agg.gitCacheCount++
}
