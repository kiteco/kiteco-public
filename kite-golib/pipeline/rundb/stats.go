package rundb

import "time"

// FeedStats describes how a particular feed was used during the pipeline's run.
type FeedStats struct {
	In           int64
	Out          int64
	ErrsByReason map[string]FeedErrors
}

// DeepCopy works as advertised
func (f FeedStats) DeepCopy() FeedStats {
	ebr := make(map[string]FeedErrors)
	for k, v := range f.ErrsByReason {
		ebr[k] = v.copy()
	}
	f.ErrsByReason = ebr
	return f
}

// Add aggregates two FeedStats objects; the receiver may be mutated.
func (f FeedStats) Add(other FeedStats) FeedStats {
	ebr := f.ErrsByReason
	if ebr == nil {
		ebr = make(map[string]FeedErrors)
	}
	for k, v := range other.ErrsByReason {
		ebr[k] = f.ErrsByReason[k].Add(v)
	}

	return FeedStats{
		In:           f.In + other.In,
		Out:          f.Out + other.Out,
		ErrsByReason: ebr,
	}
}

// FeedError describes an instance of an error for a feed
type FeedError struct {
	SourceName string
	SourceKey  string
	Error      string
	Timestamp  time.Time
}

// FeedErrors describes errors for a feed
type FeedErrors struct {
	Count   int64
	Samples []FeedError
}

// Add aggregates together two FeedErrors structs. It may mutate the receiver.
func (f FeedErrors) Add(other FeedErrors) FeedErrors {
	// If there are more samples than the limit, just take all the samples from the receiver
	// and then add the samples from other until the limit is reached.
	// This results in non-deterministic behavior, since the result depends on the order in which the errors
	// are aggregated, but that should be fine.
	samples := f.Samples
	for _, s := range other.Samples {
		if len(samples) >= maxErrorSamples {
			break
		}
		samples = append(samples, s)
	}
	f.Count += other.Count
	f.Samples = samples
	return f
}

// AddError adds an error to the stats for that error. The receiver may be mutated.
func (f FeedErrors) AddError(sourceName string, sourceKey string, err error) FeedErrors {
	f.Count++

	fe := func() FeedError {
		return FeedError{
			SourceName: sourceName,
			SourceKey:  sourceKey,
			Error:      err.Error(),
			Timestamp:  time.Now().UTC(),
		}
	}

	// reservoir sampling
	// https://en.wikipedia.org/wiki/Reservoir_sampling
	if len(f.Samples) < maxErrorSamples {
		f.Samples = append(f.Samples, fe())
		return f
	}
	keepProb := float64(maxErrorSamples) / float64(f.Count)
	if rng.Float64() >= keepProb {
		return f
	}
	idx := rng.Int() % maxErrorSamples
	f.Samples[idx] = fe()
	return f
}

func (f FeedErrors) copy() FeedErrors {
	newSamples := make([]FeedError, 0, len(f.Samples))
	for _, fe := range f.Samples {
		newSamples = append(newSamples, fe)
	}
	f.Samples = newSamples
	return f
}
