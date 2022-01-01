package main

type metricData struct {
	Labels    []int         `json:"labels"`
	NumArgs   int           `json:"num_args_selected"`
	CandComps ReturnedComps `json:"candidate_completions"`
}

// ReturnedComps contains all the necessary characteristics of completions returned by expr model.
type ReturnedComps struct {
	Completions  []int     `json:"completions"`
	NumArgsArray []int     `json:"num_args_completion"`
	Scores       []float32 `json:"scores"`
}
