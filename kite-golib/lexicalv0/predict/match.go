package predict

import (
	"reflect"

	"github.com/kiteco/kiteco/kite-golib/lexicalv0/status"
)

func matchSuffix(initialContext, context []int64) ([]int64, []int64, bool) {
	var offset int
	var overlapRange int
	var remainder []int64
	if len(context) > len(initialContext) {
		// If the query context is larger than the embedded context, we line up
		// the "front" of the two and chop off the remainder/extra context. This
		// remainder will become part of the unembedded context if there is a match
		diff := len(context) - len(initialContext)
		context, remainder = context[:len(context)-diff], context[len(context)-diff:]
		overlapRange = len(context)
	} else {
		// If the embedded context is equal or larger than the context, we line up
		// the "end" of the two by computing an offset into the embedded context
		offset = len(initialContext) - len(context)
		overlapRange = len(initialContext) - offset
	}

	// overlapRange is the number of tokens that are possible to overlap
	for i := 0; i < overlapRange; i++ {
		if reflect.DeepEqual(initialContext[offset+i:], context[:len(context)-i]) {
			var unembedded []int64
			unembedded = append(unembedded, context[len(context)-i:]...)
			unembedded = append(unembedded, remainder...)
			status.PartialRunOverlapDist.Record(int64(overlapRange - i))
			return initialContext, unembedded, true
		}
	}

	return initialContext, nil, false
}

func matchPrefix(initialContext, context []int64) int {
	i := 0
	for ; i < len(initialContext) && i < len(context); i++ {
		if initialContext[i] != context[i] {
			return i
		}
	}
	return i
}
