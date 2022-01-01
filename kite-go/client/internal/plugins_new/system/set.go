package system

import "sort"

// Union returns a union of the arrays, removing duplicates. Order of the resulting array is sorted.
func Union(arrays ...[]string) []string {
	m := make(map[string]bool)
	for _, array := range arrays {
		for _, elem := range array {
			m[elem] = true
		}
	}

	union := []string{}
	for k := range m {
		union = append(union, k)
	}
	sort.Strings(union)
	return union
}
