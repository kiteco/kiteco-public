package validation

import (
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/go-github/github"
	"github.com/kiteco/kiteco/kite-go/navigation/git"
	"github.com/kiteco/kiteco/kite-go/navigation/ignore"
	"github.com/kiteco/kiteco/kite-go/navigation/localpath"
	"github.com/kiteco/kiteco/kite-go/navigation/recommend"
)

// PullRequestID ...
type PullRequestID string

var errNoFiles = errors.New("no files edited by pull request")

// Loader ...
type Loader struct {
	PullsPath localpath.Absolute
	Root      localpath.Absolute
	Ignorer   ignore.Ignorer
}

// Load retrieves data for validating a recommender.
// the returned data is a map which associates pull requests with a slice of edited files.
// these files have blocks with line numbers based on the state of the file before being edited.
func (l Loader) Load() (map[PullRequestID][]recommend.File, error) {
	pulls := make(map[PullRequestID][]recommend.File)
	pullDirs, err := l.readPullDirs()
	if err != nil {
		return nil, err
	}
	for pull, files := range pullDirs {
		pullDir := l.PullsPath.Join(localpath.Relative(pull))
		diffBlocks, err := l.getDiffBlocks(pullDir)
		if err != nil {
			return nil, err
		}
		for _, file := range files {
			path := file.ToLocalFile(l.Root)
			if _, err := path.Lstat(); os.IsNotExist(err) {
				continue
			}
			pulls[pull] = append(pulls[pull], recommend.File{
				Path:   string(path),
				Blocks: diffBlocks[path],
			})
		}
	}
	return pulls, nil
}

func (l Loader) readPullDirs() (map[PullRequestID][]git.File, error) {
	pullDirs, err := getPullDirs(l.PullsPath)
	if err != nil {
		return nil, err
	}
	pullFiles := make(map[PullRequestID][]git.File)
	for pull, pullDir := range pullDirs {
		files, err := getPullFiles(pullDir)
		if err == errNoFiles {
			continue
		}
		if err != nil {
			return nil, err
		}
		pullFiles[pull] = files
	}
	return pullFiles, nil
}

func getPullDirs(path localpath.Absolute) (map[PullRequestID]localpath.Absolute, error) {
	numbers, err := path.Readdirnames(-1)
	if err != nil {
		return nil, err
	}
	pullDirs := make(map[PullRequestID]localpath.Absolute)
	for _, number := range numbers {
		pullDirs[PullRequestID(number)] = path.Join(number)
	}
	return pullDirs, nil
}

func getPullFiles(pullDir localpath.Absolute) ([]git.File, error) {
	fileDirs, err := getFileDirs(pullDir)
	if err != nil {
		return nil, err
	}
	var files []git.File
	for _, fileDir := range fileDirs {
		commitFile, err := getFile(fileDir)
		if err != nil {
			return nil, err
		}
		file := git.File(*commitFile.Filename)
		if !file.HasSupportedExtension() {
			continue
		}
		files = append(files, file)
	}
	return files, nil
}

func (l Loader) getDiffBlocks(pullDir localpath.Absolute) (map[localpath.Absolute][]recommend.Block, error) {
	files, err := getPullFiles(pullDir)
	if err != nil {
		return nil, err
	}
	keep := make(map[git.File]bool)
	for _, file := range files {
		keep[file] = true
	}
	fileDirs, err := getFileDirs(pullDir)
	if err != nil {
		return nil, err
	}

	diffBlocks := make(map[localpath.Absolute][]recommend.Block)
	for _, fileDir := range fileDirs {
		commitFile, err := getFile(fileDir)
		if err != nil {
			return nil, err
		}
		file := git.File(*commitFile.Filename)
		if !keep[file] {
			continue
		}
		if commitFile.Patch == nil {
			continue
		}
		blocks, err := findDiffBlocks(*commitFile.Patch)
		if err != nil {
			return nil, err
		}
		diffBlocks[file.ToLocalFile(l.Root)] = blocks
	}
	return diffBlocks, nil
}

func findDiffBlocks(patch string) ([]recommend.Block, error) {
	re := regexp.MustCompile("@@ -[0-9]+,[0-9]+ \\+[0-9]+,[0-9]+ @@")
	chunks := make(map[string][]string)
	var key string
	for _, line := range strings.Split(patch, "\n") {
		if re.MatchString(line) {
			key = line
			continue
		}
		if key == "" {
			continue
		}
		if strings.HasPrefix(line, "+") {
			continue
		}
		chunks[key] = append(chunks[key], line)
	}

	num := regexp.MustCompile("[0-9]+")
	var blocks []recommend.Block
	for data, chunk := range chunks {
		firstLine, err := strconv.Atoi(num.FindString(data))
		if err != nil {
			return nil, err
		}
		blocks = append(blocks, recommend.Block{
			Content:   strings.Join(chunk, "\n"),
			FirstLine: firstLine,
			LastLine:  firstLine + len(chunk) - 1,
		})
	}
	return blocks, nil
}

func getFile(path localpath.Absolute) (github.CommitFile, error) {
	contents, err := read(path.Join("diff.json"))
	if err != nil {
		return github.CommitFile{}, err
	}
	var commitFile github.CommitFile
	err = json.Unmarshal(contents, &commitFile)
	if err != nil {
		return github.CommitFile{}, err
	}
	return commitFile, nil
}

func getFileDirs(pullDir localpath.Absolute) ([]localpath.Absolute, error) {
	fileDirName := pullDir.Join("files")
	ids, err := fileDirName.Readdirnames(-1)
	if err != nil {
		return nil, err
	}
	var fileDirs []localpath.Absolute
	for _, id := range ids {
		fileDirs = append(fileDirs, fileDirName.Join(id))
	}
	return fileDirs, nil
}

func read(path localpath.Absolute) ([]byte, error) {
	f, err := path.Open()
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return ioutil.ReadAll(io.LimitReader(f, 1e6))
}
