package recommend

import (
	"math"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/kiteco/kiteco/kite-go/navigation/localpath"
)

type shingle uint32

const wordsRegexp = "[a-zA-Z0-9_]*"

type vectorizer struct {
	idf         map[shingle]float32
	vectorSet   vectorSet
	watchDirs   watchDirs
	wordsRegexp *regexp.Regexp
	opts        vectorizerOptions
}

type vectorSet struct {
	data map[fileID]shingleVector
	m    *sync.RWMutex
}

func newVectorSet() vectorSet {
	return vectorSet{
		data: make(map[fileID]shingleVector),
		m:    new(sync.RWMutex),
	}
}

type vectorSetChanges struct {
	updates map[fileID]shingleVector
	deletes []fileID
}

func newVectorSetChanges() vectorSetChanges {
	return vectorSetChanges{
		updates: make(map[fileID]shingleVector),
	}
}

func (d *vectorSetChanges) add(other vectorSetChanges) {
	for k, v := range other.updates {
		d.updates[k] = v
	}
	d.deletes = append(d.deletes, other.deletes...)
}

func (v vectorSet) update(c vectorSetChanges) {
	v.m.Lock()
	defer v.m.Unlock()

	for _, pathID := range c.deletes {
		delete(v.data, pathID)
	}
	for pathID, vec := range c.updates {
		v.data[pathID] = vec
	}
}

type watchDirs struct {
	data map[localpath.Absolute]time.Time
	m    *sync.Mutex
}

func newWatchDirs() watchDirs {
	return watchDirs{
		data: make(map[localpath.Absolute]time.Time),
		m:    new(sync.Mutex),
	}
}

type counter struct {
	counts          map[shingle]int
	size            int
	shingleSize     int
	keepUnderscores bool
}

type shingleCovector struct {
	coords map[shingle]float32
	norm   float32
}

type shingleVector struct {
	coords  []valuedShingle
	norm    float32
	modTime time.Time
}

type valuedShingle struct {
	shingle shingle
	value   float32
}

func (vec shingleVector) toCovector() shingleCovector {
	return shingleCovector{
		coords: toCovector(vec.coords),
		norm:   vec.norm,
	}
}

func toCovector(vec []valuedShingle) map[shingle]float32 {
	cov := make(map[shingle]float32)
	for _, coord := range vec {
		cov[coord.shingle] = coord.value
	}
	return cov
}

func toVector(cov map[shingle]float32) []valuedShingle {
	var vec []valuedShingle
	for shingle, value := range cov {
		vec = append(vec, newValuedShingle(shingle, value))
	}
	return vec
}

type vectorizerOptions struct {
	shingleSize     int
	keepUnderscores bool

	scoreRegularization float32
	probRegularization  float64

	// parameters for file and block recs based on a cursor position.
	fileLocalization  localization
	blockLocalization localization
}

func newCounter(keepUnderscores bool) counter {
	return counter{
		counts:          make(map[shingle]int),
		shingleSize:     5,
		keepUnderscores: keepUnderscores,
	}
}

func (c counter) newVectorizer() vectorizer {
	idf := make(map[shingle]float32)
	for word, count := range c.counts {
		// we use a variant idf called probabilistic idf.
		// this applies a more significant penalty to words which occur in many documents.
		// note if a word is in half the documents, then the probabilistic idf is zero.
		// https://en.wikipedia.org/wiki/tfidf
		// https://nlp.stanford.edu/IR-book/pdf/06vect.pdf
		// https://nlp.stanford.edu/IR-book/pdf/11prob.pdf
		wordIdf := float32(math.Log((float64(c.size) - float64(count)) / float64(count)))
		if wordIdf > 0 {
			idf[word] = wordIdf
		}
	}

	return vectorizer{
		idf: idf,
		opts: vectorizerOptions{
			shingleSize:         c.shingleSize,
			keepUnderscores:     c.keepUnderscores,
			scoreRegularization: 10,
			probRegularization:  0.05,
			fileLocalization: localization{
				size:   20,
				weight: 0.5,
			},
			blockLocalization: localization{
				size:   10,
				weight: 0.75,
			},
		},
		vectorSet:   newVectorSet(),
		watchDirs:   newWatchDirs(),
		wordsRegexp: regexp.MustCompile(wordsRegexp),
	}
}

func (c *counter) add(content string) {
	c.size++
	for word := range countShingles(content, c.shingleSize, c.keepUnderscores) {
		c.counts[word]++
	}
}

func (v vectorizer) recommendBlocks(base, inspect string, request Request) ([]Block, error) {
	cov, err := v.makeCovector(base, request.Location.CurrentLine, v.opts.blockLocalization)
	if err != nil {
		return nil, err
	}
	var unnormalized []Block
	var total float64
	for _, block := range splitBlocks(inspect) {
		vec := v.makeVector(block.Content)
		block.Probability = float64(v.score(cov, vec))
		if block.Probability == 0 {
			continue
		}
		total += block.Probability
		block.Keywords = v.findKeywords(block.Content, cov, request)
		unnormalized = append(unnormalized, block)
	}
	normalizer := total + v.opts.probRegularization
	var blocks []Block
	for _, block := range unnormalized {
		block.Probability /= normalizer
		blocks = append(blocks, block)
	}
	sort.Slice(blocks, func(i, j int) bool {
		if blocks[i].Probability == blocks[j].Probability {
			return blocks[i].FirstLine < blocks[j].FirstLine
		}
		return blocks[i].Probability > blocks[j].Probability
	})
	return blocks, nil
}

type localization struct {
	size   int
	weight float32
}

func (v vectorizer) makeCovector(content string, line int, local localization) (shingleCovector, error) {
	globalCovector := v.makeVector(content).toCovector()
	if line == 0 {
		return globalCovector, nil
	}
	localCovector, err := v.makeLocalCovector(content, line, local)
	if err != nil {
		return shingleCovector{}, err
	}
	return mixCovectors(globalCovector, localCovector, local), nil
}

func mixCovectors(globalCovector, localCovector shingleCovector, local localization) shingleCovector {
	mixed := make(map[shingle]float32)

	globalScale := (1 - local.weight) / globalCovector.norm
	for word, value := range globalCovector.coords {
		mixed[word] += globalScale * value
	}

	localScale := local.weight / localCovector.norm
	for word, value := range localCovector.coords {
		mixed[word] += localScale * value
	}

	return shingleCovector{
		coords: mixed,
		norm:   covectorNorm(mixed),
	}
}

func (v vectorizer) makeLocalCovector(content string, currentLine int, local localization) (shingleCovector, error) {
	curated, err := curateLocalContent(content, currentLine, local)
	if err != nil {
		return shingleCovector{}, err
	}
	return v.makeVector(curated).toCovector(), nil
}

func curateLocalContent(content string, currentLine int, local localization) (string, error) {
	// We construct content that emphasizes the text near `currentLine`.
	// The text on `currentLine` is repeated `local.size` times.
	// Text that is `k` lines above or below `currentLine` is repeated `local.size - k` times.
	// The curated content is then vectorized in the usual uniform way.
	var curated []string
	lines := strings.Split(content, "\n")
	if currentLine > len(lines) || currentLine <= 0 {
		return "", ErrInvalidCurrentLine
	}
	for i, line := range strings.Split(content, "\n") {
		// subtract 1 because `currentLine` is 1-based and `i` is 0-based
		distance := currentLine - i - 1
		if distance < 0 {
			distance *= -1
		}
		repeats := local.size - distance
		for j := 0; j < repeats; j++ {
			curated = append(curated, line)
		}
	}
	return strings.Join(curated, "\n"), nil
}

func (v vectorizer) makeVector(content string) shingleVector {
	shingles := countShingles(content, v.opts.shingleSize, v.opts.keepUnderscores)
	var size int
	for word := range shingles {
		if v.idf[word] != 0 {
			size++
		}
	}

	coords := make([]valuedShingle, size)
	i := 0
	for word, tf := range shingles {
		if v.idf[word] == 0 {
			continue
		}
		coords[i] = newValuedShingle(word, float32(math.Log(1+float64(tf)))*v.idf[word])
		i++
	}
	return shingleVector{
		coords: coords,
		norm:   vectorNorm(coords),
	}
}

func (v vectorizer) score(cov shingleCovector, vec shingleVector) float32 {
	numerator := shingleDot(cov.coords, vec.coords)
	denominator := (vec.norm + v.opts.scoreRegularization) * cov.norm
	if denominator == 0 {
		return 0
	}
	return numerator / denominator
}

func (v vectorizer) findKeywords(content string, cov shingleCovector, request Request) []Keyword {
	words := v.wordsRegexp.FindAllString(content, -1)

	var keywords []Keyword
	seen := make(map[string]bool)
	for _, word := range words {
		if seen[word] {
			continue
		}
		seen[word] = true
		vec := v.makeVector(word)
		score := float64(shingleDot(cov.coords, vec.coords))
		if score == 0 {
			continue
		}
		keywords = append(keywords, Keyword{
			Word:  word,
			Score: score,
		})
	}

	sort.Slice(keywords, func(i, j int) bool {
		if keywords[i].Score == keywords[j].Score {
			return keywords[i].Word < keywords[j].Word
		}
		return keywords[i].Score > keywords[j].Score
	})
	if request.MaxBlockKeywords != -1 && len(keywords) > request.MaxBlockKeywords {
		keywords = keywords[:request.MaxBlockKeywords]
	}
	return keywords
}

func (v vectorizer) recommendFiles(currentID fileID, content string, request Request) ([]File, error) {
	cov, err := v.makeCovector(content, request.Location.CurrentLine, v.opts.fileLocalization)
	if err != nil {
		return nil, err
	}
	return v.recommendFilesFromCovector(currentID, cov, request), nil
}

func (v vectorizer) recommendFilesFromCovector(currentID fileID, cov shingleCovector, request Request) []File {
	v.vectorSet.m.RLock()
	defer v.vectorSet.m.RUnlock()

	var files []File
	var total float64
	for id, vec := range v.vectorSet.data {
		if id == currentID {
			continue
		}
		file := File{
			id:          id,
			Probability: float64(v.score(cov, vec)),
		}
		if file.Probability == 0 {
			continue
		}
		total += file.Probability
		files = append(files, file)
	}
	normalizer := total + v.opts.probRegularization
	for i := range files {
		files[i].Probability /= normalizer
	}
	sort.Slice(files, func(i, j int) bool {
		if files[i].Probability == files[j].Probability {
			return files[i].id < files[j].id
		}
		return files[i].Probability > files[j].Probability
	})
	return files
}

func shingleDot(cov map[shingle]float32, vec []valuedShingle) float32 {
	var dot float32
	for _, coord := range vec {
		dot += cov[coord.shingle] * coord.value
	}
	return dot
}

func vectorNorm(vec []valuedShingle) float32 {
	return float32(math.Sqrt(float64(shingleDot(toCovector(vec), vec))))
}

func covectorNorm(cov map[shingle]float32) float32 {
	return float32(math.Sqrt(float64(shingleDot(cov, toVector(cov)))))
}

// split a file into blocks where the blocks are non-overlapping and separated
// by either an empty line or a line with a single character, e.g. } or {
func splitBlocks(content string) []Block {
	lines := strings.Split(content+"\n", "\n")
	var blocks []Block
	var start int
	var blockLines []string
	for curr, line := range lines {
		if len(blockLines) == 0 {
			start = curr
		}
		if len(line) > 1 {
			blockLines = append(blockLines, line)
			continue
		}
		if len(blockLines) == 0 {
			continue
		}
		if len(line) == 1 {
			blockLines = append(blockLines, line)
		}
		blocks = append(blocks, Block{
			Content:   strings.Join(blockLines, "\n"),
			FirstLine: start + 1,
			LastLine:  start + len(blockLines),
		})
		blockLines = nil
	}
	return blocks
}

func countShingles(content string, shingleSize int, keepUnderscores bool) map[shingle]int {
	counts := make(map[shingle]int)
	if !keepUnderscores {
		content = strings.ReplaceAll(content, "_", "")
	}
	lower := []rune(strings.ToLower(content))

	// we look at a sliding window in `lower` of length `shingleSize`.
	// `window` is the number of runes in the sliding window that are letters.
	var window int
	for j, r := range lower {
		if unicode.IsLetter(r) {
			window++
		}
		if j < shingleSize-1 {
			continue
		}

		// here the sliding window is `lower[j+1-shingleSize:j+1]`
		// if `window` equals `shingleSize`, then all the runes are letters.
		if window == shingleSize {
			counts[newShingle(lower[j+1-shingleSize:j+1])]++
		}
		if unicode.IsLetter(lower[j+1-shingleSize]) {
			window--
		}
	}
	return counts
}

func newShingle(rs []rune) shingle {
	// This function transforms a slice of runes into a shingle.
	// Note shingles only consist of lower case letters and we are primarily interested in a-z.
	// It might be sufficient to use a single wildcard bucket for all other lower case letters.
	// To keep things relatively simple, we use five bits per rune.
	// This leaves room for six wildcard buckets.
	//
	// Since we use 5 bits per rune and 5 runes per shingle, we are using 25 bits.
	// We could increase to 5x6 or 6x5 and still fit in a 32-bit integer.
	// If we want to increase to 5x7 or 6x6, we should switch to an 64-bit integer.
	var s shingle
	for i, r := range rs {
		if i != 0 {
			s <<= 5
		}
		if r < 'a' || r > 'z' {
			s += 26 + shingle(r%6)
			continue
		}
		s += shingle(r - 'a')
	}
	return s
}

func newValuedShingle(s shingle, v float32) valuedShingle {
	return valuedShingle{
		shingle: s,
		value:   v,
	}
}
