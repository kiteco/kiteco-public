package main

import (
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/kiteco/kiteco/kite-go/client/datadeps"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonenv"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/cmd/build/helpers"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/internal/distidx"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/internal/manifest"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonstatic"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/workerpool"
	filehelpers "github.com/kiteco/kiteco/local-pipelines/python-import-exploration/helpers"
	"github.com/spf13/cobra"
)

const analysisTimeout = 1 * time.Hour

var (
	manifestPath string
	distidxPath  string
	typeshedRoot string
	outputFile   string
	numJobs      int
	flags        = log.LstdFlags | log.Lshortfile | log.Lmicroseconds
)

func init() {
	log.SetFlags(flags)
}

func fail(err error) {
	if err != nil {
		panic(err)
	}
}

// - building

func loadAST(ctx kitectx.Context, path string, pseudopath string) (pythonstatic.ASTBundle, error) {
	f, err := os.Open(path)
	fail(err)
	defer f.Close()

	contents, err := ioutil.ReadAll(f)
	fail(err)

	opts := pythonparser.Options{}
	opts.ScanOptions.Label = pseudopath
	mod, err := pythonparser.Parse(ctx, contents, opts)
	if err != nil {
		return pythonstatic.ASTBundle{}, err
	}

	imps := pythonstatic.FindImports(ctx, pseudopath, mod)

	return pythonstatic.ASTBundle{
		AST:     mod,
		Path:    pseudopath,
		Imports: imps,
	}, nil
}

func analyze(ctx kitectx.Context, graph pythonresource.Manager, privateImports bool, roots ...string) (*pythonenv.SourceTree, error) {
	ctx.CheckAbort()

	inputs := pythonstatic.AssemblerInputs{
		Graph: graph,
	}
	opts := pythonstatic.DefaultOptions
	opts.PrivateImports = privateImports
	assembler := pythonstatic.NewAssembler(ctx, inputs, opts)

	for _, root := range roots {
		err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			ctx.CheckAbort()

			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			switch filepath.Ext(path) {
			case ".py", ".pyi":
			default:
				return nil
			}

			pseudopath, err := helpers.PseudoFilePath(root, path)
			if err != nil {
				return err
			}

			ast, err := loadAST(ctx, path, pseudopath)
			if err != nil {
				log.Printf("[ERROR] failed to parse file at pseudopath %s: %s", pseudopath, err)
				return nil
			}
			assembler.AddSource(ast)
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	assembly, err := assembler.Build(ctx)
	if err != nil {
		return nil, err
	}
	return assembly.Sources, nil
}

type buildResult struct {
	key  string
	tree *pythonenv.SourceTree
}

func extract(archive string, dir string) error {
	filename := filepath.Base(archive)
	log.Printf("Processing file : %s in dir %s\n", archive, dir)
	switch {
	case strings.HasSuffix(filename, ".tar") || strings.HasSuffix(filename, ".tgz") || strings.Contains(filename, ".tar."):
		return exec.Command("tar", "-x", "-C", dir, "-f", archive).Run()
	case strings.HasSuffix(filename, ".zip"):
		return exec.Command("unzip", "-d", dir, archive).Run()
	}
	return errors.Errorf("unknown file type for %s", filename)
}

func analyzePip(ctx kitectx.Context, graph pythonresource.Manager, reqStr string, resC chan<- buildResult) {
	ctx.CheckAbort()

	// new temp directory for each install target
	tmpDir, err := ioutil.TempDir("", "")
	fail(err)

	defer os.RemoveAll(tmpDir) // remove after the analysis is done,
	// This line can be moved at the end of the function to allow checking the content of the folder after an error

	// download source archive; this may "fail" but still download the archive, so don't check the error
	exec.Command("python3", "-m", "pip", "download", "--no-binary", ":all:", "--no-deps", "-d", tmpDir, reqStr).Run()
	// TODO(naman) also fetch dependencies?
	tmpList, err := ioutil.ReadDir(tmpDir)
	fail(err)
	switch len(tmpList) {
	case 1:
	case 0:
		log.Printf("[ERROR] analyze-pip %s: could not download source archive", reqStr)
		return
	default:
		fail(errors.Errorf("[ERROR] analyze-pip %s: more than one file downloaded by pip", reqStr))
	}
	sourceArchive := filepath.Join(tmpDir, tmpList[0].Name())

	// extract
	fail(errors.WrapfOrNil(extract(sourceArchive, tmpDir), "[ERROR] analyze-pip %s: could not extract source archive", reqStr))
	// remove archive so we can trivially get the unique project root directory with ReadDir
	fail(os.Remove(sourceArchive))
	tmpList, err = ioutil.ReadDir(tmpDir)
	if len(tmpList) > 1 {
		fail(errors.Errorf("[ERROR] analyze-pip %s: more than one file/directory in source archive", reqStr))
	}
	projRoot := filepath.Join(tmpDir, tmpList[0].Name())

	// compute the "root" for analysis
	var root string
	// try to build/install to tmpDir, simulating what pip does to install packages
	// this will hopefully handle unconventional project directory layouts etc
	// https://pip.pypa.io/en/stable/reference/pip_install/#build-system-interface
	cmd := exec.Command("python3", "setup.py", "install", "--root", tmpDir, "--no-compile")
	cmd.Dir = projRoot
	if err := cmd.Run(); err == nil {
		// find site-packages dir that was installed in tmpDir
		filepath.Walk(tmpDir, func(path string, info os.FileInfo, err error) error {
			ctx.CheckAbort()

			if path == projRoot {
				return filepath.SkipDir
			}
			if !info.IsDir() {
				return nil
			}
			switch info.Name() {
			case "site-packages", "dist-packages":
				if root != "" {
					fail(errors.Errorf("[ERROR] analyze-pip %s: found multiple install root directories", reqStr))
				}
				root = path
				return filepath.SkipDir
			}
			return nil
		})

		if root == "" {
			log.Printf("[ERROR] analyze-pip %s: could not find installed site/dist-packages directory", reqStr)
		}
	} else {
		log.Printf("[WARN] analyze-pip %s: could not run setup.py: %v", reqStr, err)
	}

	if root == "" {
		// just proceed analysis with the unbuild projectRoot
		log.Printf("[WARN] analyze-pip %s: falling back to analyzing project root", reqStr)
		root = projRoot
	}

	tree, err := analyze(ctx, graph, false, root)
	if err != nil {
		log.Printf("[ERROR] analyze-pip %s: build error %s", reqStr, err)
		return
	}

	resC <- buildResult{reqStr, tree}
	return
}

func analyzeTypeshed(ctx kitectx.Context, graph pythonresource.Manager, typeshedRoot string, resC chan<- buildResult) {
	ctx.CheckAbort()

	helper := func(subdirs ...string) *pythonenv.SourceTree {
		var roots []string
		for _, subdir := range subdirs {
			roots = append(roots, filepath.Join(typeshedRoot, subdir))
		}
		tree, err := analyze(ctx, graph, true, roots...)
		if err != nil {
			log.Printf("[ERROR] analyze-typeshed: build error %s", err)
			return nil
		}
		return tree
	}

	// we ignore the specialized 3.3, 3.4, etc versions for now, since not that much stuff lives there. TODO(naman)
	resC <- buildResult{helpers.Builtin3StubKey, helper("stdlib/3", "stdlib/2and3")}
	resC <- buildResult{helpers.ThirdParty2StubKey, helper("third_party/2", "third_party/2and3")}
	resC <- buildResult{helpers.ThirdParty3StubKey, helper("third_party/3", "third_party/2and3")}
}

func build(cmd *cobra.Command, args []string) {
	if outputFile == "" {
		log.Fatalln("no output file provided")
	}

	var reqStrs []string
	for _, arg := range args {
		f, err := os.Open(arg)
		if err != nil {
			log.Fatalln(err)
		}
		data, err := ioutil.ReadAll(f)
		if err != nil {
			log.Fatalln(err)
		}
		f.Close()

		err = filehelpers.ReadLinesWithComments(string(data), func(reqStr, comment string) error {
			if reqStr == "" {
				return nil
			}
			reqStrs = append(reqStrs, reqStr)
			return nil
		})
		if err != nil {
			log.Fatalln(err)
		}
	}
	if len(reqStrs) == 0 && typeshedRoot == "" {
		log.Fatalln("nothing to do")
	}

	datadeps.Enable()

	// TODO don't use resource manager here since the stubs can (and should) be analyzed independently.
	// this will require refinement of how to handle builtins, and the core specialized values.
	opts := pythonresource.DefaultOptions
	if manifestPath != "" {
		mF, err := os.Open(manifestPath)
		if err != nil {
			log.Fatalln(err)
		}
		opts.Manifest, err = manifest.New(mF)
		if err != nil {
			log.Fatalln(err)
		}
		mF.Close()
	}
	opts.Manifest = opts.Manifest.SymbolOnly()
	if distidxPath != "" {
		dF, err := os.Open(distidxPath)
		if err != nil {
			log.Fatalln(err)
		}
		opts.DistIndex, err = distidx.New(dF)
		if err != nil {
			log.Fatalln(err)
		}
		dF.Close()
	}
	rm, errc := pythonresource.NewManager(opts)

	if <-errc != nil {
		log.Fatalln("[FATAL] failed to init resource manager for service")
	}

	// jobs to build everything
	resC := make(chan buildResult)
	var jobs []workerpool.Job
	if typeshedRoot != "" {
		jobs = append(jobs, func() error { analyzeTypeshed(kitectx.Background(), rm, typeshedRoot, resC); return nil })
	}
	for _, reqStr := range reqStrs {
		ctx := kitectx.Background()
		closeOver := reqStr
		jobs = append(jobs, func() error {
			err := ctx.WithTimeout(analysisTimeout, func(ctx kitectx.Context) (err error) {
				ctx.CheckAbort()

				log.Printf("analyze-pip %s: started\n", closeOver)
				analyzePip(ctx, rm, closeOver, resC)
				return
			})
			if err != nil {
				if _, ok := err.(kitectx.ContextExpiredError); ok {
					log.Printf("[ERROR] analyze-pip %s: aborted\n", closeOver)
				}
			} else {
				log.Printf("analyze-pip %s: completed\n", closeOver)
			}
			return nil
		})
	}

	pool := workerpool.New(numJobs)
	pool.Add(jobs)
	go func() { // close the results channel after all jobs are complete
		pool.Wait()
		close(resC)
	}()

	out := make(map[string]*pythonenv.SourceTree)
	for res := range resC {
		if res.tree == nil {
			log.Printf("[ERROR] nil source tree for %s", res.key)
			continue
		}
		out[res.key] = res.tree
	}

	fail(helpers.StoreAnalyzed(outputFile, out))
}

// -

func init() {
	cmd.Flags().StringVarP(&manifestPath, "graph", "g", "", "symbol graph manifest path (defaults to compiled-in KiteManifest)")
	cmd.Flags().StringVarP(&distidxPath, "distidx", "d", "", "distribution index path (defaults to compiled-in KiteIndex)")
	cmd.Flags().StringVarP(&typeshedRoot, "typeshed", "t", "", "root of typeshed repository")
	cmd.Flags().StringVarP(&outputFile, "output", "o", "", "output file")
	cmd.Flags().IntVarP(&numJobs, "jobs", "j", 16, "number of jobs to execute in parallel")
}

var cmd = cobra.Command{
	Use:   "analyze --graph manifest.json -o analyzed.json.gz --typeshed typeshed/ pip-packagelist1.txt ...",
	Short: "batch analysis of stubs and pip packages",
	Args:  cobra.ArbitraryArgs,
	Run:   build,
}

func main() {
	cmd.Execute()
}
