package main

import (
	"encoding/json"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonranker"
	"github.com/kiteco/kiteco/kite-go/stackoverflow"
	"github.com/kiteco/kiteco/kite-golib/text"
)

const (
	defaultSORoot      = "/var/kite/stackoverflow"
	defaultSynonymPath = "/var/kite/stackoverflow/synonyms.json"

	defaultPackageRanker   = "model.json"
	defaultPackageFeaturer = "featurer.json"
)

var (
	brackets = regexp.MustCompile(`[<>]`)
)

func main() {
	var (
		synonymPath  string
		soRoot       string
		output       string
		rankerPath   string
		featurerPath string
	)
	flag.StringVar(&synonymPath, "syn", defaultSynonymPath, "path to the synonym file")
	flag.StringVar(&soRoot, "soRoot", defaultSORoot, "root of the so page dumps")
	flag.StringVar(&output, "out", "", "path to the output file that contains the training data")
	flag.StringVar(&rankerPath, "ranker", defaultPackageRanker, "path to the ranker (model.json)")
	flag.StringVar(&featurerPath, "feat", defaultPackageFeaturer, "path to the featurer (featurer.json)")
	flag.Parse()

	if output == "" {
		log.Fatal("must specify the path to the output file by --out")
	}

	ranker, err := pythonranker.NewPackageRankerFromJSON(rankerPath, featurerPath)

	graph, err := pythonimports.NewGraph(pythonimports.DefaultImportGraph)
	if err != nil {
		log.Fatal(err)
	}

	// packageList contains the list of package names that we are collecting data for
	packageList := ranker.Candidates()
	for i, p := range packageList {
		packageList[i] = strings.ToLower(p)
	}

	// load synonyms
	synonyms := loadSynonyms(synonymPath)

	// build a table of synonyms for package names from the map of synonyms
	packageSynonyms := buildSynonymChart(synonyms, packageList)

	detector := &detector{
		ranker:   ranker,
		graph:    graph,
		synonyms: packageSynonyms,
	}

	// data that stores the extracted training data
	data := make(map[string]map[string][]string)

	// go through all so dumps in the given directory
	log.Println("going through data...")
	files, _ := ioutil.ReadDir(soRoot)
	for _, f := range files {
		fname := path.Join(soRoot, f.Name())

		// log processing time
		start := time.Now()
		sofile, err := os.Open(fname)
		if err != nil {
			log.Fatal(err)
		}
		defer sofile.Close()

		extractData(sofile, data, detector)

		elapsed := time.Since(start)
		log.Printf("It took %s to process %s\n", elapsed, fname)
	}

	outfile, err := os.Create(output)
	if err != nil {
		log.Fatal(err)
	}
	defer outfile.Close()

	encoder := json.NewEncoder(outfile)
	encoder.Encode(data)
}

// extractData extract titles / so post contents that are relevant to a method.
func extractData(r io.Reader, data map[string]map[string][]string, detector *detector) error {
	decoder := json.NewDecoder(r)
	tokenizer := text.NewHTMLTokenizer()
	for {
		var page stackoverflow.StackOverflowPage
		err := decoder.Decode(&page)
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		id := page.GetQuestion().GetPost().GetId()
		if id < 1 {
			continue
		}

		// get tags of the post (to detect whether this post is about python and which packages are relevant)
		tagStr := page.GetQuestion().GetPost().GetTags()
		tags := strings.Split(brackets.ReplaceAllString(tagStr, " "), " ")

		// skip non-python posts
		filteredTags, isPy := detector.detectPython(tags)
		if !isPy {
			continue
		}

		title := page.GetQuestion().GetPost().GetTitle()
		// find packages that this post refers to
		detectedPackages := detector.detectPackages(filteredTags, title)
		if len(detectedPackages) == 0 {
			continue
		}

		for _, p := range detectedPackages {
			if _, exists := data[p]; !exists {
				data[p] = make(map[string][]string)
			}

			var seenMethods []string

			// get all the posts in the page
			soPosts := page.GetAnswers()
			soPosts = append(soPosts, page.GetQuestion())

			// detect which methods are mentioned in the answer
			for _, post := range soPosts {
				code := text.CodeTokensFromHTML(post.GetPost().GetBody())
				foundMethods := detector.detectFuncs(p, strings.Join(code, " "))

				content := tokenizer.Tokenize(post.GetPost().GetBody())
				for _, m := range foundMethods {
					data[p][m] = append(data[p][m], content...)
				}
				seenMethods = append(seenMethods, foundMethods...)
			}
			seenMethods = text.Uniquify(seenMethods)
			for _, m := range seenMethods {
				data[p][m] = append(data[p][m], title)
			}
		}
	}
	return nil
}
