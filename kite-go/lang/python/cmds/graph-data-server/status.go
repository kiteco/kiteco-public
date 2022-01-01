package main

import (
	"github.com/kiteco/kiteco/kite-golib/status"
)

var (
	mainSection         = status.NewSection("/python/graph-data-server")
	fetchHashDuration   = mainSection.SampleDuration("fetch hash")
	fetchSourceDuration = mainSection.SampleDuration("fetch source")
	getInputsDuration   = mainSection.SampleDuration("get sample inputs")

	exprSection             = status.NewSection("/python/graph-data-server infer-expr")
	exprTrainSampleDuration = exprSection.SampleDuration("build infer-expr train sample")
	exprGraphsPerSample     = exprSection.CounterDistribution("graphs per infer-expr sample")

	sessionSection               = status.NewSection("/python/graph-data-server sessions")
	getSessionBatchDuration      = sessionSection.SampleDuration("get batch for session")
	encodeSessionBatchDuration   = sessionSection.SampleDuration("encode batch for session")
	transferSessionBatchDuration = sessionSection.SampleDuration("transfer batch for session")
)
