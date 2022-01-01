//go:generate go-bindata -pkg main -o bindata.go scripts
package main

import (
	"unicode"

	"github.com/kiteco/kiteco/kite-go/response"
)

type completionsResponse struct {
	Completions []response.SandboxCompletion `json:"completions"`
}

type reducedSandboxCompletion struct {
	Display           string `json:"display"`
	Insert            string `json:"insert"`
	Hint              string `json:"hint"`
	Synopsis          string `json:"synopsis"`
	DocumentationLink string `json:"web_docs_link"`
}

const (
	maxCompletionsPerResp = 7

	completionTypeRegular  = "regular"
	completionTypeXXXXXXX = "XXXXXXX"
)

func getInitialIdx(data []byte) (idx int64) {
	var inCommentMode bool
	for _, b := range data {
		idx++
		if inCommentMode {
			if b == '\n' {
				inCommentMode = false
			}
		} else {
			if b == '#' {
				inCommentMode = true
			} else if !unicode.IsSpace(rune(b)) {
				return idx
			}
		}
	}
	return idx
}

func main() {
	panic("code path is no longer supported.")

	// var outDir string
	// var outName string
	// var completionsType string
	// flag.StringVar(&outDir, "out", os.TempDir(), "directory to output json files to")
	// flag.StringVar(&outName, "name", "completions.json", "desired filename for the output")
	// flag.StringVar(&completionsType, "type", completionTypeRegular, "type of completion to precache")
	// flag.Parse()
	// // make sure type is legal
	// var legalType bool
	// switch completionsType {
	// case completionTypeRegular:
	// case completionTypeXXXXXXX:
	// 	legalType = true
	// }
	// if !legalType {
	// 	log.Fatalf("input type %s is not recognized", completionsType)
	// }

	// //for each script
	// scripts, err := AssetDir("scripts")
	// if err != nil {
	// 	log.Fatalf("Error loading sample file names: %v", err)
	// 	return
	// }

	// // load python services
	// pythonOpts := python.DefaultServiceOptions
	// services, err := websandbox.LoadServices(&pythonOpts)
	// if err != nil {
	// 	log.Fatalf("Error loading python services: %v", err)
	// 	return
	// }

	// scriptMap := make(map[string]map[string][]*reducedSandboxCompletion)

	// for _, script := range scripts {
	// 	completionsForScript := make(map[string][]*reducedSandboxCompletion)
	// 	scriptData, err := Asset(path.Join("scripts", script))
	// 	if err != nil {
	// 		log.Fatalf("Error loading sample file %s: %v", script, err)
	// 		return
	// 	}
	// 	var byteIdx int //getInitialIdx(scriptData)
	// 	for byteIdx < len(scriptData) {
	// 		var completionsSlice []*reducedSandboxCompletion
	// 		payload := websandbox.CompletionRequest{
	// 			Text:        string(scriptData[:byteIdx]),
	// 			CursorBytes: int64(byteIdx),
	// 			Filename:    "",
	// 			ID:          "machine",
	// 		}

	// 		switch completionsType {
	// 		case completionTypeXXXXXXX:

	// 			resp, expected, err := websandbox.XXXXXXXCompletions(kitectx.Background(), services, payload)
	// 			// conditions to retry
	// 			switch {
	// 			case err != nil:
	// 				log.Printf("Error getting completions: %v", err)
	// 				continue
	// 			case (resp == nil || resp.Completions == nil) && expected:
	// 				time.Sleep(500 * time.Millisecond)
	// 				// TODO: hacky af
	// 				if byteIdx >= 105 && byteIdx <= 110 {
	// 					// skip "title" since there is a bug in the completions
	// 					// expected logic that returns true for this case
	// 					break
	// 				}
	// 				if byteIdx >= 120 && byteIdx <= 128 {
	// 					// skip "filename" since there is a bug in the completions
	// 					// expected logic that returns true for this case
	// 					break
	// 				}
	// 				log.Println("Completions not found but expected")
	// 				continue
	// 			case expected && len(resp.Completions) == 0:
	// 				time.Sleep(500 * time.Millisecond)
	// 				log.Println("Completions empty, but expected")
	// 				continue
	// 			}
	// 			if resp != nil && resp.Completions != nil {
	// 				for i, completion := range resp.Completions {
	// 					if i >= maxCompletionsPerResp {
	// 						break
	// 					}
	// 					completionsSlice = append(completionsSlice, &reducedSandboxCompletion{
	// 						Display:           completion.Display,
	// 						Insert:            completion.Text, // TODO(dane): Is this right?
	// 						Hint:              completion.Hint,
	// 						Synopsis:          completion.Synopsis,
	// 						DocumentationLink: completion.WebDocsLink, // TODO(dane): Is this right?
	// 					})
	// 				}
	// 			}
	// 		default:
	// 			resp, expected, err := websandbox.Completions(kitectx.Background(), services, payload)
	// 			// conditions to retry
	// 			switch {
	// 			case err != nil:
	// 				log.Printf("Error getting completions: %v", err)
	// 				continue
	// 			case (resp == nil || resp.Completions == nil) && expected:
	// 				time.Sleep(500 * time.Millisecond)
	// 				// TODO: hacky af
	// 				if byteIdx >= 105 && byteIdx <= 110 {
	// 					// skip "title" since there is a bug in the completions
	// 					// expected logic that returns true for this case
	// 					break
	// 				}
	// 				if byteIdx >= 120 && byteIdx <= 128 {
	// 					// skip "filename" since there is a bug in the completions
	// 					// expected logic that returns true for this case
	// 					break
	// 				}
	// 				log.Println("Completions not found but expected")
	// 				continue
	// 			case expected && len(resp.Completions) == 0:
	// 				time.Sleep(500 * time.Millisecond)
	// 				log.Println("Completions empty, but expected")
	// 				continue
	// 			}
	// 			if resp != nil && resp.Completions != nil {
	// 				for i, completion := range resp.Completions {
	// 					if i >= maxCompletionsPerResp {
	// 						break
	// 					}
	// 					completionsSlice = append(completionsSlice, &reducedSandboxCompletion{
	// 						Display: completion.Display,
	// 						Insert:  completion.Insert,
	// 						Hint:    completion.Hint,
	// 					})
	// 				}
	// 			}
	// 		}

	// 		completionsForScript[strconv.Itoa(byteIdx)] = completionsSlice
	// 		byteIdx++
	// 		log.Printf("Completed:\n%s\n", string(scriptData[:byteIdx]))
	// 		time.Sleep(500 * time.Millisecond)
	// 	}
	// 	scriptMap[script] = completionsForScript
	// }
	// //write json to file; report size
	// buf, err := json.Marshal(scriptMap)
	// if err != nil {
	// 	log.Fatalf("Error marshalling completions to json: %v", err)
	// 	return
	// }
	// err = ioutil.WriteFile(path.Join(outDir, outName), buf, os.ModePerm)
	// if err != nil {
	// 	log.Fatalf("Error writing JSON to file: %s", err)
	// 	return
	// }

	// info, err := os.Stat(path.Join(outDir, outName))
	// if err == nil {
	// 	log.Printf("FILESIZE: %d", info.Size())
	// }

	// log.Printf("Success! json output to %s", outDir)
}
