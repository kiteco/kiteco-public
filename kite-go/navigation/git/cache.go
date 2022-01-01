package git

import (
	"encoding/json"
	"errors"

	"github.com/kiteco/kiteco/kite-go/navigation/metrics"
)

type repoBundle struct {
	Repos         map[repoKey]repoCache
	EvictionOrder []repoKey
}

type repoKey string

type repoCache struct {
	Commits  map[CommitHash][]int
	Files    []File
	inverted map[File]int
}

func newRepoCache() repoCache {
	return repoCache{
		Commits:  make(map[CommitHash][]int),
		inverted: make(map[File]int),
	}
}

func (r repoCache) get(hash CommitHash) (Commit, bool, error) {
	fileIDs, ok := r.Commits[hash]
	if !ok {
		return Commit{}, false, nil
	}
	var files []File
	for _, fileID := range fileIDs {
		if fileID >= len(r.Files) {
			return Commit{}, false, errors.New("Invalid file ID")
		}
		files = append(files, r.Files[fileID])
	}
	recovered := Commit{
		Hash:  hash,
		Files: files,
	}
	return recovered, true, nil
}

func (r *repoCache) add(hash CommitHash, commit Commit) {
	var ids []int
	for _, file := range commit.Files {
		if id, ok := r.inverted[file]; ok {
			ids = append(ids, id)
			continue
		}
		id := len(r.Files)
		r.inverted[file] = id
		r.Files = append(r.Files, file)
		ids = append(ids, id)
	}
	r.Commits[hash] = ids
}

func newRepoBundle() repoBundle {
	return repoBundle{
		Repos: make(map[repoKey]repoCache),
	}
}

func (b repoBundle) get(key repoKey) (repoCache, bool) {
	repo, ok := b.Repos[key]
	if !ok {
		return newRepoCache(), false
	}
	repo.inverted = make(map[File]int)
	for id, file := range repo.Files {
		repo.inverted[file] = id
	}
	return repo, true
}

func (b *repoBundle) add(key repoKey, cache repoCache) {
	if b.Repos == nil {
		b.Repos = make(map[repoKey]repoCache)
	}
	b.Repos[key] = cache

	var newEvictionOrder []repoKey
	for _, k := range b.EvictionOrder {
		if k == key {
			continue
		}
		newEvictionOrder = append(newEvictionOrder, k)
	}
	newEvictionOrder = append(newEvictionOrder, key)
	b.EvictionOrder = newEvictionOrder
}

func unmarshalRepoBundle(data []byte) (repoBundle, error) {
	var b repoBundle
	err := json.Unmarshal(data, &b)
	if err != nil {
		return repoBundle{}, err
	}
	return b, nil
}

func (b *repoBundle) evictAndMarshal(maxSize int64) ([]byte, bool, error) {
	original, err := json.MarshalIndent(b, "", "")
	if err != nil {
		return nil, false, err
	}
	if len(original) <= int(maxSize) {
		return original, false, nil
	}

	// Cache misses are expensive, since processing a lot of commits is expensive.
	// For this reason, we really do not want to ever evict.
	// But if we never evict, the cache data file could grow without bound,
	// because the user could work in an arbitrary number of git repos.
	// The cache data file is approximately 1 MB per repo,
	// so for many users, we could fit all their repos in the cache without ever evicting.
	// Ultimately, we evict to guarantee the size of the cache is bounded,
	// but we assume evicting is uncommon.
	err = b.evict(len(original) - int(maxSize))
	if err != nil {
		return nil, false, err
	}
	data, err := json.MarshalIndent(b, "", "")
	if err != nil {
		return nil, false, err
	}
	return data, true, nil
}

func (b *repoBundle) evict(excess int) error {
	var cleared int
	for cleared < excess && len(b.EvictionOrder) > 0 {
		victim := b.EvictionOrder[0]
		victimBytes, err := json.MarshalIndent(b.Repos[victim], "", "")
		if err != nil {
			return err
		}
		delete(b.Repos, victim)
		b.EvictionOrder = b.EvictionOrder[1:]
		cleared += len(victimBytes)
	}
	return nil
}

type hitEvict struct {
	hit   bool
	evict bool
}

func (b repoBundle) logCacheMetrics(numBytes int, h hitEvict) {
	var hits int64
	if h.hit {
		hits = 1
	}

	var evictions int64
	if h.evict {
		evictions = 1
	}

	var numCommits, numFiles int
	for _, repo := range b.Repos {
		numCommits += len(repo.Commits)
		numFiles += len(repo.Files)
	}

	gitCacheMetrics := metrics.GitCache{
		NumRepos:   int64(len(b.Repos)),
		NumCommits: int64(numCommits),
		NumFiles:   int64(numFiles),
		NumBytes:   int64(numBytes),
		Hits:       hits,
		Evictions:  evictions,
	}
	gitCacheMetrics.Log()
}
