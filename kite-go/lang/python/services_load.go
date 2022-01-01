package python

import (
	"fmt"
	"log"
	"runtime/debug"
	"time"

	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/kiteco/kiteco/kite-go/lang/python/answers"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonbatch"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncode"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncuration"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonindex"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonmodels"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonskeletons"
	"github.com/kiteco/kiteco/kite-go/lang/python/seo"
	"github.com/kiteco/kiteco/kite-go/localcode"
	"github.com/pkg/errors"
)

func (s *Services) timer(start time.Time, name string) {
	log.Printf("[python.Services] %s %s", time.Since(start), name)
}

func (s *Services) initResourceManager(opts ServiceOptions) error {
	defer s.timer(time.Now(), "resource manager")
	rmOpts := opts.ResourceManager
	rm, errc := pythonresource.NewManager(rmOpts)
	if err := <-errc; err != nil {
		return errors.Wrap(err, "failed to init resource manager for service")
	}
	s.ResourceManager = rm
	debug.FreeOSMemory()
	return nil
}

func (s *Services) loadImportGraph(opts ServiceOptions) error {
	defer s.timer(time.Now(), "import graph")

	var err error
	s.ImportGraph, err = pythonimports.NewGraph(opts.ImportGraph)
	if err != nil {
		return fmt.Errorf("error loading python import graph: %s", err)
	}

	defer s.timer(time.Now(), "update import graph with skeletons")
	if err = pythonskeletons.UpdateGraph(s.ImportGraph); err != nil {
		return fmt.Errorf("error updating graph with python skeletons: %v", err)
	}

	debug.FreeOSMemory()
	return nil
}

func (s *Services) loadAnswers(opts ServiceOptions) error {
	defer s.timer(time.Now(), "Answers")

	if opts.AnswersPath != "" {
		answers, err := answers.Load(opts.AnswersPath)
		s.Answers = answers
		return err
	}
	return nil
}

func (s *Services) loadSEO(opts ServiceOptions) error {
	defer s.timer(time.Now(), "SEO")

	if opts.SEOPath != "" {
		data, err := seo.Load(opts.SEOPath)
		s.SEOData = data
		return err
	}
	return nil
}

func (s *Services) loadCuration(graph *pythonimports.Graph, opts ServiceOptions) error {
	defer s.timer(time.Now(), "curation")

	if (opts.Curation != pythoncuration.SearchOptions{}) {
		var err error
		s.Curation, err = pythoncuration.NewSearcher(graph, &opts.Curation)
		if err != nil {
			return fmt.Errorf("error loading python curated examples: %v", err)
		}
		debug.FreeOSMemory()
	}

	return nil
}

func (s *Services) setupLocalcode(opts ServiceOptions) error {
	defer s.timer(time.Now(), "localcode setup")

	s.BuilderLoader = &pythonbatch.BuilderLoader{
		Graph:   s.ResourceManager,
		Options: opts.Batch,
	}

	localcode.RegisterLoader(lang.Python, s.BuilderLoader.Load)

	return nil
}

func (s *Services) loadInvertedIndex(manager pythonresource.Manager, curation *pythoncuration.Searcher, opts ServiceOptions) error {
	defer s.timer(time.Now(), "inverted index")

	if (opts.Index != pythonindex.ClientOptions{}) {
		if curation == nil {
			log.Println("warning: not loading inverted index because curation is not loaded")
			return nil
		}

		s.InvertedIndex = pythonindex.NewClient(manager, pythoncode.DefaultPackageStats, curation.AllCurated(), &opts.Index)
		debug.FreeOSMemory()
	}

	return nil
}

func (s *Services) loadModels(opts ServiceOptions) error {
	defer s.timer(time.Now(), "models")

	var err error
	s.Models, err = pythonmodels.New(opts.ModelOptions)
	if err != nil {
		return fmt.Errorf("error loading models: %v", err)
	}
	return nil
}
