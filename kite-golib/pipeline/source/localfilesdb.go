package source

import (
	"bufio"
	"fmt"
	"log"
	"math/rand"
	"sort"
	"strconv"
	"strings"

	"github.com/kiteco/kiteco/kite-go/localfiles"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
	"github.com/kiteco/kiteco/kite-golib/pipeline"
	"github.com/kiteco/kiteco/kite-golib/pipeline/sample"

	// postgres driver
	_ "github.com/lib/pq"
)

const (
	// WestUS2UserMachineIndex is the path to the user-machine index created from the westus2 localfiles DB
	WestUS2UserMachineIndex = "s3://kite-local-pipelines/python-localfiles-index/2019-08-11-18-02-37-15-PM/users.csv"

	idxRandomSeed = 3
)

// LocalFilesDBConfig ...
type LocalFilesDBConfig struct {
	DBURI            string
	S3Bucket         string
	S3Region         string
	UserMachineIndex string
	MaxRecords       int
}

// NewLocalFilesDB returns a Source that iterates through the user/machine pairs that are present in the
// local files DB and emits sample.Corpus instances representing all of the files that have been uploaded for a given
// user/machine pair.
func NewLocalFilesDB(conf LocalFilesDBConfig) (pipeline.Source, error) {
	idx, err := newUserMachineIndex(conf.UserMachineIndex)
	if err != nil {
		return nil, fmt.Errorf("error reading user machine index: %v", err)
	}
	// shuffle the index such as to not emit corpora in order of user ID
	idx.Shuffle(idxRandomSeed)

	db := localfiles.FileDB("postgres", conf.DBURI)

	opts := localfiles.DefaultContentStoreOptions
	opts.BucketName = conf.S3Bucket
	opts.Region = conf.S3Region
	store, err := localfiles.NewContentStore(opts, db)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize local files store: %v", err)
	}

	return &localFilesDB{
		idx:   idx,
		conf:  conf,
		store: store,
		max:   conf.MaxRecords,
	}, nil
}

type localFilesDB struct {
	idx   userMachineIndex
	conf  LocalFilesDBConfig
	store *localfiles.ContentStore
	count int
	max   int
}

func (l *localFilesDB) Name() string {
	return "LocalFilesSource"
}

func (l *localFilesDB) ForShard(shard, totalShards int) (pipeline.Source, error) {
	if totalShards != 1 {
		return nil, fmt.Errorf("distributed mode not supported")
	}
	return l, nil
}

func (l *localFilesDB) SourceOut() pipeline.Record {
	if l.count >= l.max {
		return pipeline.Record{}
	}
	umc := l.idx[l.count%len(l.idx)]
	l.count++

	var s pipeline.Sample
	var err error
	s, err = l.newLocalFilesCorpus(umc.UserID, umc.Machine)
	if err != nil {
		s = pipeline.WrapError("error getting corpus", err)
	}

	return pipeline.Record{
		Key:   fmt.Sprintf("%d-%s", umc.UserID, umc.Machine),
		Value: s,
	}
}

type userMachineCount struct {
	UserID  int64
	Machine string
	Count   int64
}

type userMachineIndex []userMachineCount

func newUserMachineIndex(path string) (userMachineIndex, error) {
	f, err := fileutil.NewCachedReader(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var idx []userMachineCount

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		toks := strings.Split(line, ",")
		if len(toks) != 3 {
			return nil, fmt.Errorf("expected 3 tokens, got %d for line: %s", len(toks), line)
		}

		uid, err := strconv.ParseInt(toks[0], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("error parsing user ID from line: %s", line)
		}
		machine := toks[1]
		count, err := strconv.ParseInt(toks[2], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("error parsing count from line: %s", line)
		}

		umc := userMachineCount{
			UserID:  uid,
			Machine: machine,
			Count:   count,
		}

		idx = append(idx, umc)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error scanning through %s", path)
	}

	if len(idx) == 0 {
		return nil, fmt.Errorf("empty index read from %s", path)
	}

	log.Printf("read %d users/machines from %s", len(idx), path)
	return idx, nil
}

func (u userMachineIndex) Shuffle(seed int64) {
	r := rand.New(rand.NewSource(seed))
	r.Shuffle(len(u), func(i, j int) { u[i], u[j] = u[j], u[i] })
}

type localFilesCorpus struct {
	userID  int64
	machine string
	store   *localfiles.ContentStore

	files []sample.FileInfo
	// map of filenames to their respective hashes
	hashes map[string]string
}

func (l *localFilesDB) newLocalFilesCorpus(userID int64, machine string) (localFilesCorpus, error) {
	files, err := l.store.Files.List(userID, machine)
	if err != nil {
		return localFilesCorpus{}, err
	}

	fis := make([]sample.FileInfo, 0, len(files))
	hashes := make(map[string]string)
	for _, f := range files {
		fis = append(fis, sample.FileInfo{
			Name:      f.Name,
			CreatedAt: f.CreatedAt,
			UpdatedAt: f.UpdatedAt,
		})
		hashes[f.Name] = f.HashedContent
	}

	sort.Slice(fis, func(i, j int) bool {
		return fis[i].Name < fis[j].Name
	})

	return localFilesCorpus{
		userID:  userID,
		machine: machine,
		store:   l.store,
		files:   fis,
		hashes:  hashes,
	}, nil

}

func (localFilesCorpus) SampleTag() {}

func (l localFilesCorpus) ID() string {
	return fmt.Sprintf("%d-%s", l.userID, l.machine)
}

func (l localFilesCorpus) List() ([]sample.FileInfo, error) {
	return l.files, nil
}

func (l localFilesCorpus) Get(filename string) ([]byte, error) {
	hash, found := l.hashes[filename]
	if !found {
		return nil, fmt.Errorf("file not found: %s", filename)
	}

	return l.store.Get(hash)
}
