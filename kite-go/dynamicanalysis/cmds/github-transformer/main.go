package main

import (
	"compress/gzip"
	"encoding/json"
	"flag"
	"io"
	"log"
	"os"

	"github.com/kiteco/kiteco/kite-go/dynamicanalysis"
)

func main() {
	var input, output string
	flag.StringVar(&input, "input", "", "json.gz file containing output of runtime type inference (ASTs in json format)")
	flag.StringVar(&output, "output", "", "json.gz file that will contain map of fqn to return types")
	flag.Parse()

	if input == "" {
		log.Fatalf("Please specify input file name.")
	}

	decoder := dynamicanalysis.NewDecoder(input)

	returnTypes := make(map[string][]string)
	for {
		dat := make(map[string]interface{})
		err := decoder.Decode(&dat)
		if err != nil {
			if err == io.ErrUnexpectedEOF {
				log.Println("Encountered unexpected EOF")
				break
			} else if err == io.EOF {
				log.Println("Reached EOF. Exit gracefully.")
				break
			} else {
				log.Fatal(err)
				break
			}
		}

		var callExprs []interface{}
		dynamicanalysis.CollectValuesForKey("Call", dat["RootArray"], &callExprs)

		for _, call := range callExprs {
			callMap, ok := call.(map[string]interface{})
			if !ok {
				log.Println("Call expr should be a map")
				continue
			}

			log.Println("Finding fully qualified name...")
			fullyQualified, err := dynamicanalysis.FindFullyQualifiedName(callMap)
			if err != nil {
				log.Println(err)
				continue
			}
			if fullyQualified == "" {
				log.Printf("Fully qualified name is empty for: \n%v.", callMap)
				continue
			}

			log.Println("Finding return type...")
			returnType := ""
			ret, ok := callMap["k_type"]
			if ok {
				if returnType, ok = ret.(string); !ok {
					log.Println("Return type should be a string")
					continue
				}
			}

			returnTypes[fullyQualified] = append(returnTypes[fullyQualified], returnType)
			log.Printf("Function %s returns %s\n", fullyQualified, returnType)
		}
	}

	// create a file for output
	fout, err := os.Create(output)
	if err != nil {
		log.Fatal(err)
	}
	defer fout.Close()

	// create a compressor and an encoder
	comp := gzip.NewWriter(fout)
	enc := json.NewEncoder(comp)
	defer comp.Close()

	enc.Encode(returnTypes)
	log.Printf("Num expressions evaluated: %d\n", len(returnTypes))
}
