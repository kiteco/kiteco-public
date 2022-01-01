package main

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"flag"
	"log"
	"os"
	"strconv"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	"github.com/kiteco/kiteco/kite-go/annotate"
	"github.com/kiteco/kiteco/kite-go/curation"
	"github.com/kiteco/kiteco/kite-go/curation/segment"
	"github.com/kiteco/kiteco/kite-go/dynamicanalysis"
	"github.com/kiteco/kiteco/kite-go/lang"
	_ "github.com/lib/pq"
)

func main() {
	var outpath, dockerimage, pkgFilter, langFilter, snapshotids string
	var annotations, runResults, approvedOnly bool
	var limit int
	var ID int64
	flag.StringVar(&outpath, "output", "", "path to which to write output")
	flag.BoolVar(&annotations, "annotations", false, "execute code examples during dumping")
	flag.BoolVar(&runResults, "runs", true, "retrieve runs for code examples and inclue in output")
	flag.BoolVar(&approvedOnly, "approvedOnly", false, "only dumps approved examples")
	flag.IntVar(&limit, "limit", 0, "limit the number of code examples to output")
	flag.StringVar(&dockerimage, "dockerimage", "", "docker image in which to run examples")
	flag.StringVar(&pkgFilter, "package", "", "only dump snippets for this package (requires --language)")
	flag.StringVar(&langFilter, "language", "", "only dump snippets for this language (requires --package)")
	flag.StringVar(&snapshotids, "snapshotids", "", "snapshot ids for snippets to be dumped (requries --snapshotids)")
	flag.Int64Var(&ID, "id", 0, "dump this snapshot ID only")
	flag.Parse()

	log.SetFlags(log.LstdFlags | log.Lshortfile)

	db := curation.GormDB(os.Getenv("CODEEXAMPLE_DB_DRIVER"), os.Getenv("CODEEXAMPLE_DB_URI"))
	runs := curation.NewRunManager(db)
	manager := curation.NewCuratedSnippetManager(db, runs)

	var err error
	var snippets []*curation.CuratedSnippet
	if snapshotids != "" {
		ids, err := readSnapshotIDs(snapshotids)
		if err != nil {
			log.Fatal(err)
		}

		seen := make(map[int64]struct{})
		for i, id := range ids {
			if i%1000 == 0 {
				log.Printf("Processed %d out of %d ranking labels\n", i, len(ids))
			}
			if _, exists := seen[id]; !exists {
				seen[id] = struct{}{}
				snip, err := manager.GetBySnapshotID(id)
				if err != nil {
					log.Printf("cannot process snapshot id %d: %v\n", id, err)
					continue
				}
				snippets = append(snippets, snip)
			}
		}

	} else if ID > 0 {
		snip, err := manager.GetBySnapshotID(ID)
		if err != nil {
			log.Fatalf("cannot process snapshot id %d: %v\n", ID, err)
		}
		snippets = append(snippets, snip)

	} else if pkgFilter != "" || langFilter != "" {
		if pkgFilter == "" {
			log.Fatalln("If you provide --language then you must also provide --package")
		}
		if langFilter == "" {
			log.Fatalln("If you provide --package then you must also provide --langage")
		}
		snippets, err = manager.List(langFilter, pkgFilter)
		if err != nil {
			log.Fatalln("Error fetching snippets from database:", err)
		}
		if len(snippets) == 0 {
			log.Fatalf("No snippets in database for language %s, package %s", langFilter, pkgFilter)
		}
	} else {
		snippets, err = manager.ListAll()
		if err != nil {
			log.Fatalln("Error fetching snippets from database:", err)
		}
	}

	// Only include approved examples if the flag is set
	var approved []*curation.CuratedSnippet
	if approvedOnly {
		for _, snippet := range snippets {
			if snippet.Status == curation.SnippetStatusApproved {
				approved = append(approved, snippet)
			}
		}
		snippets = approved
	}

	// Apply limit
	if limit > 0 && len(snippets) > limit {
		log.Printf("Processing %d of %d snippets\n", limit, len(snippets))
		snippets = snippets[:limit]
	}

	// Open the output stream
	var f *os.File
	if outpath == "" {
		f = os.Stdout
	} else {
		f, err = os.Create(outpath)
		if err != nil {
			log.Fatalln(err)
		}
		defer f.Close()
	}
	gzipper := gzip.NewWriter(f)
	defer gzipper.Close()

	w := json.NewEncoder(gzipper)

	// Fetch the runs
	var numWritten, numFailed, numMissing, numUnknownLanguage, numEmptyCode, numEmptyTitle int
	for _, snippet := range snippets {
		if strings.HasPrefix(snippet.Package, "kite_") {
			continue
		}

		log.Printf("\n\nDumping '%s' (SnapshotID=%d)\n", snippet.Title, snippet.SnapshotID)

		if snippet.Title == "" {
			log.Printf("Warning: missing title for snippet %d. Will not be included in output.\n", snippet.SnippetID)
			numEmptyTitle++
			continue
		}
		if snippet.Code == "" {
			log.Printf("Warning: missing code for snippet %d. Will not be included in output.\n", snippet.SnippetID)
			numEmptyCode++
			continue
		}
		language := lang.FromName(snippet.Language)
		if language == lang.Unknown {
			log.Printf("Unknown language %s. Skipping this example.\n", snippet.Language)
			numUnknownLanguage++
			continue
		}

		// Fill in ApparatusSpec from the postlude
		snippet.ApparatusSpec, err = annotate.ExtractSpec(language, snippet.Postlude)
		if err != nil {
			log.Println("error extracting apparatus spec:", err)
		}

		// Get the latest run for this snippet
		example := curation.Example{
			Snippet: snippet,
		}
		if runResults {
			example.Result, err = runs.LookupLatestForSnippetAggregate(snippet.SnippetID)
			if err != nil {
				log.Fatalf("error looking up latest run for snippet %d: %v\n", snippet.SnippetID, err)
			}
			if example.Result == nil {
				// There is no output for this snippet
				log.Printf("Warning: no run found for snippet %d. Will not be included in output.\n", snippet.SnippetID)
				numMissing++
				continue
			}
			if !example.Result.Run.Succeeded {
				// This code example failed to execute
				log.Printf("Warning: latest run for snippet %d failed. Will not be included in output.\n", snippet.SnippetID)
				numFailed++
				continue
			}
		}

		if annotations {
			if len(dockerimage) == 0 {
				log.Fatalln("Cannot run code example with no docker image specified")
			}

			regions := curation.RegionsFromSnippet(snippet)

			// Run once in presentation mode
			flow, err := annotate.RunWithRegions(regions, snippet.Postlude, annotate.Options{
				Language:    language,
				DockerImage: dockerimage,
			})
			if err != nil {
				log.Printf("Error processing code example: %v. Skipping this example.\n", err)
				numFailed++
				continue
			}
			if !flow.Raw.Succeeded {
				log.Println("Error executing code example. Skipping this example:")
				log.Println(string(flow.Raw.Stdout))
				log.Println(string(flow.Raw.Stderr))
				log.Println(flow.Raw.SandboxError.Error())
				numFailed++
				continue
			}

			example.Result.Segments = curation.SegmentsFromAnnotations(flow.Segments)
			example.Result.InputFiles = flow.InputFiles

			// Run again using dynamic analysis
			references, flow, err := dynamicanalysis.TraceSnippetReferences(snippet, dynamicanalysis.DefaultTraceOptions)
			if err != nil {
				log.Printf("Error tracing references: %v. Skipping this example.\n", err)
				goto Write
			}
			if !flow.Raw.Succeeded {
				log.Println("Error tracing references. Skipping this example:")
				log.Println(string(flow.Raw.Stdout))
				log.Println(string(flow.Raw.Stderr))
				log.Println(flow.Raw.SandboxError.Error())
				goto Write
			}

			// Get references traced from dynamic analysis
			if len(references) == 0 {
				goto Write
			}
			indexed := dynamicanalysis.GetLineIndexed(references, flow.Stencil.Runnable)
			dynamicanalysis.SortReferencesByLineNumber(indexed)

			// Filter out unpresentable code and transform indices
			linesReferenced := make(map[int]int)
			for i, j := range flow.Stencil.LineMap {
				linesReferenced[j] = i
			}

			var codeSegs []*curation.Segment
			counts := [][]int{}
			for _, seg := range example.Result.Segments {
				if seg.Type == segment.Code {
					codeSegs = append(codeSegs, seg)
					lines := strings.Split(seg.Code, "\n")
					count := []int{0}
					for _, line := range lines {
						count = append(count, count[len(count)-1]+len(line)+1)
					}
					counts = append(counts, count)
				}
			}
			if len(codeSegs) == 0 {
				goto Write
			}

			refIdx := 0
			for segIdx, seg := range codeSegs {
				count := counts[segIdx]
				for ; refIdx < len(indexed); refIdx++ {
					i := indexed[refIdx]
					if l, exists := linesReferenced[i.LineNumber]; exists {
						if l < seg.BeginLineNumber {
							continue
						}
						if l > seg.EndLineNumber {
							break
						}
						begin := count[l-seg.BeginLineNumber] + i.ColOffset
						end := begin + i.Reference.Length()
						seg.References = append(seg.References, curation.Reference{
							Begin:              begin,
							End:                end,
							Original:           i.Reference.Original,
							FullyQualifiedName: i.Reference.FullyQualifiedName,
							Instance:           i.Reference.Instance,
							NodeType:           i.Reference.NodeType,
						})
					}
				}
			}
		}

	Write:
		err = w.Encode(example)
		if err != nil {
			log.Fatalln(err)
		}
		numWritten++
	}

	log.Printf("Wrote %d snippets and ignored:\n", numWritten)
	log.Printf("  %d that failed to execute\n", numFailed)
	log.Printf("  %d with no output\n", numMissing)
	log.Printf("  %d with unknown language\n", numUnknownLanguage)
	log.Printf("  %d with no title\n", numEmptyTitle)
	log.Printf("  %d with no code\n", numEmptyCode)
}

func readSnapshotIDs(path string) ([]int64, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var ids []int64
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		id, err := strconv.ParseInt(scanner.Text(), 10, 64)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, scanner.Err()
}
