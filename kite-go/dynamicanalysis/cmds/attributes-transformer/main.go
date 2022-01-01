package main

import (
	"compress/gzip"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/kiteco/kiteco/kite-go/dynamicanalysis"
)

type setOfStrings map[string]struct{}

type attrsByType map[string]setOfStrings
type attrsBySnippet map[string]attrsByType

type resultByType map[string][]string
type resultBySnippet map[string]resultByType

func main() {
	var input, output string
	var perSnippet bool
	flag.StringVar(&input, "input", "", "json.gz file containing output of runtime type inference (ASTs in json format)")
	flag.StringVar(&output, "output", "", "json.gz file that will contain map of type to attributes called on it")
	flag.BoolVar(&perSnippet, "perSnippet", false, "specify if result should be attributes per snippet, false if it should be aggregate")
	flag.Parse()

	if input == "" {
		log.Fatalf("Please specify input file name.")
	}

	decoder := dynamicanalysis.NewDecoder(input)

	attributes := make(attrsBySnippet)
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

		// Get all Attribute nodes from AST
		var attributeNodes []interface{}
		dynamicanalysis.CollectValuesForKey("Attribute", dat["RootArray"], &attributeNodes)

		// Get snippet hash as string
		snippetHash, err := getStringValue(dat, "hash")
		if err != nil {
			log.Fatal(err)
		}

		byType := make(attrsByType)
		for _, attributeNode := range attributeNodes {
			log.Println("Finding attributes called on type...")
			typ, attr, err := findTypeAndAttribute(attributeNode)
			if err != nil {
				log.Println(err)
				continue
			}

			if byType[typ] == nil {
				byType[typ] = make(setOfStrings)
			}
			byType[typ][attr] = struct{}{}
		}
		attributes[snippetHash] = byType
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

	if perSnippet {
		res := make(resultBySnippet)
		for hash, byType := range attributes {
			res[hash] = make(resultByType)
			for typ, attrs := range byType {
				for attr := range attrs {
					res[hash][typ] = append(res[hash][typ], attr)
				}
			}
		}
		enc.Encode(res)
		log.Printf("Output per snippet. %d snippets.\n", len(res))
	} else {
		aggregate := make(resultByType)
		for _, byType := range attributes {
			for typ, attrs := range byType {
				for attr := range attrs {
					aggregate[typ] = append(aggregate[typ], attr)
				}
			}
		}
		enc.Encode(aggregate)
		log.Printf("Output aggregate. %d unique types.", len(aggregate))
	}
}

func findTypeAndAttribute(node interface{}) (string, string, error) {
	attributeNode, ok := node.(map[string]interface{})
	if !ok {
		return "", "", fmt.Errorf("`Attribute` node should be a map")
	}

	valueNode, err := getMapValue(attributeNode, "value")
	if err != nil {
		return "", "", err
	}
	if valueNode == nil {
		return "", "", fmt.Errorf("`Attribute` node must have `value` node")
	}

	// Find base type
	// Eg. for the expression os.path.join(), get `module.path`
	// Eg. for the expression bytearray.remove(), get `bytearray`
	var typ string
	var errForType error
	relevantNodes := []string{"Attribute", "Call", "Name"}
	for _, n := range relevantNodes {
		if node, ok := valueNode[n]; ok {
			typ, errForType = getStringValue(node, "k_type")
			if typ == "module" {
				typ, errForType = getStringValue(node, "id")
			}
		}
	}
	if errForType != nil {
		return "", "", errForType
	}
	if typ == "" {
		return "", "", fmt.Errorf("Couldn't find base type")
	}

	// Find attribute
	attr, err := getStringValue(attributeNode, "attr")
	if err != nil {
		return "", "", err
	}
	if attr == "" {
		return "", "", fmt.Errorf("Couldn't find attr")
	}

	return typ, attr, nil
}

func getStringValue(node interface{}, key string) (string, error) {
	val, err := getValue(node, key)
	if err != nil {
		return "", err
	}
	if val == nil {
		return "", nil // it's fine for a node to not have a key
	}
	valStr, ok := val.(string)
	if !ok {
		return "", fmt.Errorf("val must be string")
	}
	return valStr, nil
}

func getMapValue(node interface{}, key string) (map[string]interface{}, error) {
	val, err := getValue(node, key)
	if err != nil {
		return nil, err
	}
	if val == nil {
		return nil, nil // it's fine for a node to not have a key
	}
	valMap, ok := val.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("val must be map")
	}
	return valMap, nil
}

func getValue(node interface{}, key string) (interface{}, error) {
	nodeData, ok := node.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("Node must be map")
	}
	val, ok := nodeData[key]
	if !ok {
		return nil, nil // it's fine for a node to not have a key
	}
	return val, nil
}
