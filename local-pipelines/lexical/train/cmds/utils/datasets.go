package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0"
)

// DedupedCrawlForLang returns the deduped crawl dataset location given language
func DedupedCrawlForLang(l lang.Language) string {
	switch l {
	case lang.Golang:
		return DedupedGoCrawl
	case lang.JavaScript:
		return DedupedJSCrawl
	case lang.JSX:
		return DedupedJSXCrawl
	case lang.Vue:
		return DedupedVueCrawl
	case lang.Python:
		return DedupedPythonCrawl
	}
	return ""
}

// ShuffledCrawlForLang returns the deduped crawl dataset location given language
func ShuffledCrawlForLang(l lang.Language) string {
	switch l {
	case lang.Golang:
		return ShuffledGoCrawl
	case lang.JavaScript:
		return ShuffledJSCrawl
	case lang.JSX:
		return ShuffledJSXCrawl
	case lang.Vue:
		return ShuffledVueCrawl
	case lang.Python:
		return ShuffledPythonCrawl
	}
	return ""
}

// DatasetForLang ..
func DatasetForLang(t DatasetType, group lexicalv0.LangGroup) []string {
	var roots []string
	for _, l := range group.Langs {
		switch l {
		case lang.Golang:
			roots = append(roots, GoSplitRoot)
		case lang.JavaScript:
			roots = append(roots, JSSplitRoot)
		case lang.Vue:
			roots = append(roots, VueSplitRoot)
		case lang.JSX:
			roots = append(roots, JSXSplitRoot)
		case lang.Python:
			roots = append(roots, PythonSplitRoot)
		default:
			for _, ext := range lang.LanguageTags[l].Exts {
				if supportedTextExt(ext) {
					roots = append(roots, TextSplitRootForExt(ext))
				} else {
					panic(fmt.Sprintf("WARNING: lang %s with ext %s does not have text data", l.Name(), ext))
				}
			}
		}
	}

	seen := make(map[string]bool)
	var deduped []string
	for _, root := range roots {
		if seen[root] {
			continue
		}
		seen[root] = true
		deduped = append(deduped, root)
	}
	roots = deduped

	var paths []string
	for _, root := range roots {
		paths = append(paths, fileutil.Join(root, string(t)))
	}

	return paths
}

// DatasetsForLang ...
func DatasetsForLang(group lexicalv0.LangGroup, ts ...DatasetType) []string {
	var ds []string
	for _, t := range ts {
		ds = append(ds, DatasetForLang(t, group)...)
	}
	return ds
}

// LocalFilesDataset walks the list of directories contained in content
// and collects all of the local files that are compatible with the provided language.
func LocalFilesDataset(group lexicalv0.LangGroup, content string) ([]string, error) {
	var dirs []string
	for _, l := range strings.Split(content, "\n") {
		if l := strings.TrimSpace(l); l != "" {
			dirs = append(dirs, l)
		}
	}

	var files []string
	for _, d := range dirs {
		err := filepath.Walk(d, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return errors.Errorf("error iterating file %s in %s with error: %s", path, d, err)
			}
			if info.IsDir() {
				return nil
			}
			// 128kb filter
			if info.Size() > (1 << 17) {
				return nil
			}
			if strings.Contains(path, "test") {
				return nil
			}

			if !group.Contains(lang.FromFilename(path)) {
				return nil
			}

			// Go filters
			if strings.HasSuffix(path, "bindata.go") || strings.HasSuffix(path, "pb.go") {
				return nil
			}
			// JS Filters
			if strings.Contains(path, "node_modules") || strings.HasSuffix(path, ".min.js") {
				return nil
			}
			// Python Filters
			if strings.Contains(path, "kite_ml/env") || strings.Contains(path, "kite_ml/tfenv") {
				return nil
			}

			files = append(files, path)
			return nil
		})
		if err != nil {
			return nil, errors.Errorf("error walking: %v", err)
		}
	}
	sort.Strings(files)

	return files, nil
}

// LocalFilesKiteco returns a list of files located on disk that are compatible with the
// given language. localDataRoot is assumed to be the root of the kiteco repo.
// TODO: kind of nasty, should we just upload a copy of kiteco to s3?
func LocalFilesKiteco(group lexicalv0.LangGroup, localDataRoot string) ([]string, error) {
	if localDataRoot == "" {
		// try getting local data root from gopath
		localDataRoot = os.Getenv("GOPATH")
		if localDataRoot != "" {
			localDataRoot = filepath.Join(localDataRoot, "src/github.com/kiteco/kiteco")
		}
	}

	if localDataRoot == "" {
		return nil, errors.Errorf("no localDataRoot set")
	}

	var dirs []string
	for _, l := range group.Langs {
		switch l {
		case lang.Golang:
			dirs = []string{
				filepath.Join(localDataRoot, "kite-golib"),
				filepath.Join(localDataRoot, "kite-go"),
			}
		case lang.JavaScript:
			dirs = []string{
				filepath.Join(localDataRoot, "sidebar"),
				filepath.Join(localDataRoot, "kite-answers"),
				filepath.Join(localDataRoot, "web"),
			}
		case lang.Python:
			dirs = []string{
				filepath.Join(localDataRoot, "kite-python"),
			}
		}
	}

	files, err := LocalFilesDataset(group, strings.Join(dirs, "\n"))
	if err != nil {
		return nil, err
	}

	if len(files) == 0 {
		return nil, errors.Errorf("unable to find any files in %v", dirs)
	}

	return files, nil
}

// LocalFiles returns a list of files located on disk that are compatible with the
// given language. Default is kiteco repo.
func LocalFiles(group lexicalv0.LangGroup, localDataRoot string) ([]string, error) {
	if localDataRoot == "" {
		return LocalFilesKiteco(group, localDataRoot)
	}
	files, err := LocalFilesDataset(group, strings.Join([]string{localDataRoot}, "\n"))
	if err != nil {
		return nil, err
	}

	if len(files) == 0 {
		return nil, errors.Errorf("unable to find any files in %v", localDataRoot)
	}
	return files, nil
}
