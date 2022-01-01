package tfserving

import (
	"context"
	"strings"
	"time"

	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	serving_proto "github.com/kiteco/kiteco/kite-golib/protobuf/tensorflow/serving"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"sync/atomic"
)

var globalMetrics *Metrics

func init() {
	globalMetrics = &Metrics{}
}

// GetMetrics returns global tfserving client metrics
func GetMetrics() *Metrics {
	return globalMetrics
}

// Metrics tracks the number requests and timeouts
type Metrics struct {
	requests  uint64
	timeouts  uint64
	othererrs uint64
	success   uint64

	queuefull         uint64
	connectionclosed  uint64
	connectionrefused uint64
	otherconnerr      uint64
	otherunavailable  uint64
}

func (m *Metrics) hit(err error) {
	atomic.AddUint64(&m.requests, 1)
	switch {
	case status.Code(err) == codes.Unavailable:
		msg := err.Error()
		if strings.Contains(msg, "connection closed") {
			atomic.AddUint64(&m.connectionclosed, 1)
		} else if strings.Contains(msg, "connection refused") {
			atomic.AddUint64(&m.connectionrefused, 1)
		} else if strings.Contains(msg, "batch scheduling queue") {
			atomic.AddUint64(&m.queuefull, 1)
		} else if strings.Contains(msg, "connection error") {
			atomic.AddUint64(&m.otherconnerr, 1)
		} else {
			atomic.AddUint64(&m.otherunavailable, 1)
		}
	case status.Code(err) == codes.DeadlineExceeded || err == context.DeadlineExceeded:
		atomic.AddUint64(&m.timeouts, 1)
	case err != nil:
		atomic.AddUint64(&m.othererrs, 1)
	default:
		atomic.AddUint64(&m.success, 1)
	}
}

// MetricsSnapshot is a by-value snapshot of requested metrics
type MetricsSnapshot struct {
	Requests  uint64
	Timeouts  uint64
	OtherErrs uint64
	Success   uint64

	// Breaking down "Unavailable" status code
	QueueFull         uint64
	ConnectionClosed  uint64
	ConnectionRefused uint64
	OtherConnErr      uint64
	OtherUnavailable  uint64
}

// Read returns the current count and optionally clears them
func (m *Metrics) Read(clear bool) MetricsSnapshot {
	if clear {
		return MetricsSnapshot{
			Requests:  atomic.SwapUint64(&m.requests, 0),
			Timeouts:  atomic.SwapUint64(&m.timeouts, 0),
			OtherErrs: atomic.SwapUint64(&m.othererrs, 0),
			Success:   atomic.SwapUint64(&m.success, 0),

			QueueFull:         atomic.SwapUint64(&m.queuefull, 0),
			ConnectionClosed:  atomic.SwapUint64(&m.connectionclosed, 0),
			ConnectionRefused: atomic.SwapUint64(&m.connectionrefused, 0),
			OtherConnErr:      atomic.SwapUint64(&m.otherconnerr, 0),
			OtherUnavailable:  atomic.SwapUint64(&m.otherunavailable, 0),
		}
	}

	return MetricsSnapshot{
		Requests:  atomic.LoadUint64(&m.requests),
		Timeouts:  atomic.LoadUint64(&m.timeouts),
		OtherErrs: atomic.LoadUint64(&m.othererrs),
		Success:   atomic.LoadUint64(&m.success),

		QueueFull:         atomic.LoadUint64(&m.queuefull),
		ConnectionClosed:  atomic.LoadUint64(&m.connectionclosed),
		ConnectionRefused: atomic.LoadUint64(&m.connectionrefused),
		OtherConnErr:      atomic.LoadUint64(&m.otherconnerr),
		OtherUnavailable:  atomic.LoadUint64(&m.otherunavailable),
	}
}

const timeout time.Duration = 2 * time.Second

func doRequest(kctx kitectx.Context, client serving_proto.PredictionServiceClient, req *serving_proto.PredictRequest) (*serving_proto.PredictResponse, error) {
	var resp *serving_proto.PredictResponse

	errc := kitectx.Go(func() error {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		var err error
		resp, err = client.Predict(ctx, req)
		GetMetrics().hit(err)
		return err
	})

	select {
	case <-kctx.AbortChan():
		kctx.Abort()
		// NOTE: this error will never be returned since Abort will panic
		// internally to implement the abort logic.
		return nil, errors.New("request cancelled")
	case err := <-errc:
		if err != nil {
			return nil, err
		}
		return resp, nil
	}
}
