package localcode

import (
	"sync"

	"github.com/kiteco/kiteco/kite-golib/status"
)

var (
	workerSection = status.NewSection("localcode (Worker)")

	loopDuration        = workerSection.SampleDuration("do")
	listFilesDuration   = workerSection.SampleDuration("do (list files)")
	builderDuration     = workerSection.SampleDuration("do (builder)")
	builderErrorRatio   = workerSection.Ratio("do (builder error rate)")
	listFilesErrorRatio = workerSection.Ratio("do (list files error rate)")
	noNewFilesRatio     = workerSection.Ratio("do (no new files)")
	builderTimeoutRatio = workerSection.Ratio("do (builder timeout)")

	fileGetDuration      = workerSection.SampleDuration("localFileGetter (Get)")
	filePrefetchDuration = workerSection.SampleDuration("localFileGetter (Prefetch)")

	fileGetCacheRatio = workerSection.Ratio("localFileGetter (diskcache hit rate)")
	fileGetFoundRatio = workerSection.Ratio("localFileGetter (file returned rate)")

	putDuration       = workerSection.SampleDuration("localPutter (Put)")
	putWriterDuration = workerSection.SampleDuration("localPutter (PutWriter)")

	handleRequestsDuration     = workerSection.SampleDuration("requestServer (handleRequests)")
	requestToSelectedDuration  = workerSection.SampleDuration("requestServer (first requested to selected)")
	requestToCompletedDuration = workerSection.SampleDuration("requestServer (first requested to completed)")
	queueSizeCount             = workerSection.SampleInt64("requestServer (queue size)")

	findMatchingDuration   = workerSection.SampleDuration("artifactServer (find matching artifacts)")
	handleArtifactDuration = workerSection.SampleDuration("artifactServer (serve artifact)")
)

var (
	clientSection = status.NewSection("localcode (Client)")

	artifactStatusBreakdown = clientSection.Breakdown("artifactClient (request status codes)")

	firstArtifactDuration = clientSection.SampleDuration("UserContext (first request to first artifact)")
	allArtifactDuration   = clientSection.SampleDuration("UserContext (request to artifact)")
	perLoaderDuration     = clientSection.SampleDuration("UserContext (loader, per load)")
	servedDirtyDuration   = clientSection.SampleDuration("UserContext (how long dirty artifact was served)")
	loaderErrorRatio      = clientSection.Ratio("UserContext (loader error rate)")
	usingDirtyArtifact    = clientSection.Ratio("UserContext (returning dirty artifact)")
	neverGotArtifact      = clientSection.Ratio("UserContext (never got artifact)")
	neverMadeRequest      = clientSection.Ratio("UserContext (never made request)")

	indexWithErroredFile = clientSection.Ratio("UserContext (got index containing errored file)")

	haveIndexRatio                 = clientSection.Ratio("UserContext with index")
	hadIndexWithRequestRatio       = clientSection.Ratio("UserContext (ever had index) / (made request)")
	haveIndexWithRequestRatio      = clientSection.Ratio("UserContext (currently has index) / (made request)")
	haveErrorWithRequestRatio      = clientSection.Ratio("UserContext (has only error) / (made request)")
	haveDirtyIndexWithRequestRatio = clientSection.Ratio("UserContext (index dirty) / (made request & has index)")

	cleanupDuration = clientSection.SampleDuration("cleanup")
)

func init() {
	hadIndexWithRequestRatio.Headline = true
	firstArtifactDuration.Headline = true

	builderErrorRatio.Headline = true
	builderTimeoutRatio.Headline = true
	listFilesErrorRatio.Headline = true
	loopDuration.Headline = true
	requestToSelectedDuration.Headline = true
	requestToCompletedDuration.Headline = true
}

var (
	// This section keeps track of per-artifact-name statistics. We use a RWLock
	// to minimize lock contention.
	// TODO(tarak): The methods in status that return these objects should be using this
	// locking method. Its easier to use it here for now since its the only use-case AFAIK.
	rw                sync.RWMutex
	artifactSection   = status.NewSection("localcode (Artifacts)")
	artifactBytes     = map[string]*status.SampleBytes{}
	artifactDurations = map[string]*status.SampleDuration{}
)

func artifactDuration(name string) *status.SampleDuration {
	rw.RLock()

	var exists bool
	var d *status.SampleDuration
	if d, exists = artifactDurations[name]; !exists {
		rw.RUnlock()

		rw.Lock()
		defer rw.Unlock()
		// Check again since object could have been added between RUnlock & Lock
		if d, exists = artifactDurations[name]; !exists {
			d = artifactSection.SampleDuration(name)
			artifactDurations[name] = d
		}
		return d
	}

	rw.RUnlock()
	return d
}

func artifactByte(name string) *status.SampleBytes {
	rw.RLock()

	var exists bool
	var b *status.SampleBytes
	if b, exists = artifactBytes[name]; !exists {
		rw.RUnlock()

		rw.Lock()
		defer rw.Unlock()
		// Check again since object could have been added between RUnlock & Lock
		if b, exists = artifactBytes[name]; !exists {
			b = artifactSection.SampleByte(name)
			artifactBytes[name] = b
		}
		return b
	}

	rw.RUnlock()
	return b
}
