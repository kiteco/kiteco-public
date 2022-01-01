package main

import (
	"context"
	"log"
	"time"

	"github.com/kiteco/kiteco/kite-golib/errors"
	tf_proto "github.com/kiteco/kiteco/kite-golib/protobuf/tensorflow/core/framework"
	serving_proto "github.com/kiteco/kiteco/kite-golib/protobuf/tensorflow/serving"
	"google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

type server struct {
	conn        *grpc.ClientConn
	client      serving_proto.PredictionServiceClient
	logAll      bool
	logErrs     bool
	contextSize int
}

func newServer(forwardTo string, logAll, logErrs bool, contextSize int) (*server, error) {
	conn, err := grpc.Dial(forwardTo, grpc.WithInsecure())
	if err != nil {
		return nil, errors.Wrapf(err, "cannot connect to tfserving grpc server")
	}

	client := serving_proto.NewPredictionServiceClient(conn)

	return &server{
		conn:        conn,
		client:      client,
		logAll:      logAll,
		logErrs:     logErrs,
		contextSize: contextSize,
	}, nil
}

func (m *server) Predict(ctx context.Context, request *serving_proto.PredictRequest) (*serving_proto.PredictResponse, error) {
	start := time.Now()

	var langBundle metricBundle

	name := request.GetModelSpec().GetName()
	switch name {
	case "go-large":
		name = "golang"
		langBundle = goBundle
	case "py-large":
		name = "python"
		langBundle = pyBundle
	case "js-large":
		name = "javascript"
		langBundle = jsBundle
	case "all-langs-large":
		name = "all-langs"
		langBundle = allBundle
	default:
		muxBundle.otherErrs.Add(1)
		err := status.Errorf(codes.InvalidArgument, "unrecognized model %s",
			request.GetModelSpec().GetName())
		if m.logErrs || m.logAll {
			go log.Printf("[tfserving-mux] model: %s, err: %s, took: %s", name, err.Error(), time.Since(start))
		}
		return nil, err
	}

	modelBreakdown.Hit(name)

	muxBundle.requests.Add(1)
	langBundle.requests.Add(1)

	muxBundle.inflight.Add(1)
	defer muxBundle.inflight.Add(-1)

	langBundle.inflight.Add(1)
	defer langBundle.inflight.Add(-1)

	defer muxBundle.durations.DeferRecord(time.Now())
	defer langBundle.durations.DeferRecord(time.Now())

	var resized bool
	if m.contextSize > 0 {
		resizeErr := m.resizeContext(request)
		if resizeErr == nil {
			resized = true
		}
	}

	resp, err := m.client.Predict(ctx, request)
	if err != nil {
		switch {
		case err == context.Canceled || status.Code(err) == codes.Canceled:
			muxBundle.canceled.Add(1)
			langBundle.canceled.Add(1)
		case err == context.DeadlineExceeded || status.Code(err) == codes.DeadlineExceeded:
			muxBundle.deadlined.Add(1)
			langBundle.deadlined.Add(1)
		default:
			muxBundle.otherErrs.Add(1)
			langBundle.otherErrs.Add(1)
		}
		if m.logAll || m.logErrs {
			go log.Printf("[tfserving-mux] model: %s, resized_context: %t (%d), err: %s, took: %s", name, resized, m.contextSize, err.Error(), time.Since(start))
		}
	} else if m.logAll {
		go log.Printf("[tfserving-mux] model: %s, resized_context: %t (%d), took: %s", name, resized, m.contextSize, time.Since(start))
	}

	return resp, err
}

func (m *server) resizeContext(request *serving_proto.PredictRequest) error {
	inputs := request.GetInputs()

	contextTensor := inputs["context"]
	if contextTensor == nil {
		return errors.Wrapf(nil, "could not find input tensor 'context'")
	}
	paddedContext := contextTensor.Int64Val

	contextMaskTensor := inputs["context_mask"]
	if contextMaskTensor == nil {
		return errors.Wrapf(nil, "could not find input tensor 'context_mask'")
	}
	contextMask := contextMaskTensor.Int64Val

	// Extract the unpadded context
	var context []int64
	for i := 0; i < len(contextMask); i++ {
		if contextMask[i] == 1 {
			context = paddedContext[i:]
			break
		}
	}

	// If the context is larger than target size, truncate from the left
	if len(context) > m.contextSize {
		context = context[len(context)-m.contextSize:]
	}

	paddedContext, contextMask = padContext(context, m.contextSize, 0)

	inputs["context"] = contextPlaceholder(paddedContext)
	inputs["context_mask"] = contextPlaceholder(contextMask)

	return nil
}

func contextPlaceholder(context []int64) *tf_proto.TensorProto {
	return &tf_proto.TensorProto{
		Dtype: tf_proto.DataType_DT_INT64,
		TensorShape: &tf_proto.TensorShapeProto{
			Dim: []*tf_proto.TensorShapeProto_Dim{
				&tf_proto.TensorShapeProto_Dim{
					Size: int64(1),
				},
				&tf_proto.TensorShapeProto_Dim{
					Size: int64(len(context)),
				},
			},
		},
		Int64Val: context,
	}
}

// Copied over from predict package to avoid importing tensorflow dependency

// padContext pads the context with `padValue` such that it is the provided length. It returns
// the padded context and a mask over the padded context corresponding to the added padding.
// e.g [1,2,3,4] w/ window 6 -> [padValue,padValue,1,2,3,4], [0,0,1,1,1,1]
func padContext(context []int64, window int, padValue int64) ([]int64, []int64) {
	if len(context) >= window {
		return context, ones(window, 0)
	}

	nPad := window - len(context)

	padding := make([]int64, nPad, nPad+len(context))
	if padValue != 0 {
		for i := range padding {
			padding[i] = padValue
		}
	}

	padding = append(padding, context...)
	return padding, ones(window, nPad)
}

func ones(size, offset int) []int64 {
	v := make([]int64, size)
	for i := 0; i < len(v); i++ {
		if i >= offset {
			v[i] = 1
		}
	}
	return v
}
