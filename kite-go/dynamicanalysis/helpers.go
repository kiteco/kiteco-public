package dynamicanalysis

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
)

// FindFullyQualifiedName extracts the fully qualified name of the given Call
// expression, as annotated by the tracing code.
func FindFullyQualifiedName(callMap map[string]interface{}) (string, error) {
	// Find fully qualified name of the expression
	fqn := ""
	names, err := CollectStringsForKey("k_function_fqn", callMap["func"])
	if err != nil {
		return "", fmt.Errorf("error while type asserting method fully qualified names to strings: %v", err)
	}
	if len(names) > 0 {
		fqn = names[0]
	}

	if fqn == "" {
		names, err := CollectStringsForKey("k_instance_class_fqn", callMap["func"])
		if err != nil {
			return "", fmt.Errorf("error while type asserting class fully qualified names to strings: %v", err)
		}
		if len(names) > 0 {
			fqn = names[0]
		}
	}

	return fqn, nil
}

func sliceOfStrings(original []interface{}) ([]string, error) {
	var strings []string
	for i, elem := range original {
		elemStr, ok := elem.(string)
		if !ok {
			return nil, fmt.Errorf("element %d is not string but %T", i, elem)
		}
		strings = append(strings, elemStr)
	}
	return strings, nil
}

// CollectStringsForKey is a generic helper that collects all values for the given
// key from the given tree, and type asserts them to strings before returning.
func CollectStringsForKey(key string, root interface{}) ([]string, error) {
	var buf []interface{}
	CollectValuesForKey(key, root, &buf)
	return sliceOfStrings(buf)
}

type byK []string

func (b byK) Len() int {
	return len(b)
}

func (b byK) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}

// All strings starting with the prefix 'k_' are greater than all other strings.
// Resort to lexicographical sort otherwise.
func (b byK) Less(i, j int) bool {
	var ret bool
	if strings.HasPrefix(b[j], "k_") && !strings.HasPrefix(b[i], "k_") {
		ret = true
	} else if strings.HasPrefix(b[i], "k_") && !strings.HasPrefix(b[j], "k_") {
		ret = false
	} else {
		ret = b[i] < b[j] // lexicographical
	}
	return ret
}

// CollectValuesForKey is a generic helper that collects all values for the given name
// from the given tree, populating the vals slice passed in by reference.
func CollectValuesForKey(name string, root interface{}, vals *[]interface{}) {
	switch a := root.(type) {
	case map[string]interface{}:
		// Sort keys to ensure a map is always traversed in the same order of keys
		// First the normal ast keys in lexicographical order, followed by the keys
		// with 'k_' as a prefix in lexicographical order. This is important because
		// maps don't maintain order so each traversal would yield a different order
		// which is undesirable.
		var keys []string
		for k := range a {
			keys = append(keys, k)
		}

		sort.Sort(byK(keys))

		for _, k := range keys {
			v := a[k]
			if k == name {
				if vMap, ok := v.(map[string]interface{}); ok {
					*vals = append(*vals, copyMap(vMap))
				} else if vStr, ok := v.(string); ok {
					*vals = append(*vals, vStr)
				} else if vFloat, ok := v.(float64); ok {
					*vals = append(*vals, vFloat)
				}
			}
			CollectValuesForKey(name, v, vals)
		}
	case []interface{}:
		for _, v := range a {
			CollectValuesForKey(name, v, vals)
		}
	}
}

// NewDecoder returns a new json decoder to read from the given file.
func NewDecoder(path string) *json.Decoder {
	in, err := os.Open(path)
	if err != nil {
		log.Fatalf("Error opening %s: %v", path, err)
	}

	decomp, err := gzip.NewReader(in)
	if err != nil {
		log.Fatalf("Error gunzipping %s: %v", path, err)
	}

	return json.NewDecoder(decomp)
}

func copyMap(m map[string]interface{}) map[string]interface{} {
	cpy := make(map[string]interface{})
	for k, v := range m {
		cpy[k] = v
	}
	return cpy
}
