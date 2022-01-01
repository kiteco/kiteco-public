package main

import (
	"testing"

	"github.com/stretchr/testify/require"

	tf_proto "github.com/kiteco/kiteco/kite-golib/protobuf/tensorflow/core/framework"
	serving_proto "github.com/kiteco/kiteco/kite-golib/protobuf/tensorflow/serving"
)

func Test_ResizeContext(t *testing.T) {
	s := &server{}

	type testCase struct {
		contextSize     int
		paddedSize      int
		resizeTo        int
		expectedContext []int64
		expectedMask    []int64
	}

	cases := []testCase{
		{
			// no-op
			contextSize:     5,
			paddedSize:      5,
			resizeTo:        5,
			expectedContext: []int64{1, 2, 3, 4, 5},
			expectedMask:    []int64{1, 1, 1, 1, 1},
		},
		{
			// resize -> larger
			contextSize:     5,
			paddedSize:      5,
			resizeTo:        7,
			expectedContext: []int64{0, 0, 1, 2, 3, 4, 5},
			expectedMask:    []int64{0, 0, 1, 1, 1, 1, 1},
		},
		{
			// resize -> smaller
			contextSize:     5,
			paddedSize:      5,
			resizeTo:        3,
			expectedContext: []int64{3, 4, 5},
			expectedMask:    []int64{1, 1, 1},
		},
		{
			// resize -> larger, w/ existing padding
			contextSize:     5,
			paddedSize:      7,
			resizeTo:        10,
			expectedContext: []int64{0, 0, 0, 0, 0, 1, 2, 3, 4, 5},
			expectedMask:    []int64{0, 0, 0, 0, 0, 1, 1, 1, 1, 1},
		},
		{
			// no context (e.g, all padding) -> larger
			contextSize:     0,
			paddedSize:      5,
			resizeTo:        10,
			expectedContext: []int64{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			expectedMask:    []int64{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		},
		{
			// no context (e.g, all padding) -> smaller
			contextSize:     0,
			paddedSize:      5,
			resizeTo:        3,
			expectedContext: []int64{0, 0, 0},
			expectedMask:    []int64{0, 0, 0},
		},
	}

	for _, tc := range cases {
		req := makeRequest(tc.contextSize, tc.paddedSize)
		s.contextSize = tc.resizeTo
		err := s.resizeContext(req)
		require.NoError(t, err)
		require.Equal(t, tc.expectedContext, req.GetInputs()["context"].Int64Val)
		require.Equal(t, tc.expectedMask, req.GetInputs()["context_mask"].Int64Val)
	}

}

func makeRequest(contextSize, paddedSize int) *serving_proto.PredictRequest {
	context := make([]int64, contextSize)
	for i := 0; i < contextSize; i++ {
		context[i] = int64(i + 1)
	}

	context, mask := padContext(context, paddedSize, 0)

	inputs := make(map[string]*tf_proto.TensorProto)
	inputs["context"] = contextPlaceholder(context)
	inputs["context_mask"] = contextPlaceholder(mask)

	return &serving_proto.PredictRequest{
		Inputs: inputs,
	}
}
