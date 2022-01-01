package recommend

import (
	"sort"
)

var defaultGraphOptions = graphOptions{
	regularization: 1,
	editWeight:     0.01,
}

// A commit or pull request.
type commitID uint32

// A graph which we use to analyze the relationship between files and commits.
// We build a bipartite graph whose nodes represent files and commits.
// Edges connect commits to the files they modify.
type graph struct {
	files    map[fileID][]commitID
	editSize map[commitID]uint32

	editScores     map[fileID]float32
	totalEditScore float32

	opts graphOptions
}

type graphOptions struct {
	regularization float32
	editWeight     float32
}

func (g graph) recommendFiles(base fileID) []File {
	scores, total := g.computeScores(base)
	normalizer := total + g.opts.regularization
	var recs []File
	for inspect, score := range scores {
		if inspect == base {
			continue
		}
		recs = append(recs, File{
			id:          inspect,
			Probability: float64(score / normalizer),
		})
	}

	sort.Slice(recs, func(i, j int) bool {
		if recs[i].Probability == recs[j].Probability {
			return recs[i].id < recs[j].id
		}
		return recs[i].Probability > recs[j].Probability
	})
	return recs
}

// We combine two scoring functions, one based on edits and one based on co-edits.
func (g graph) computeScores(base fileID) (map[fileID]float32, float32) {
	scores, coeditTotal := g.computeCoeditScores(base)
	if scores == nil {
		return g.editScores, g.totalEditScore
	}
	for file, score := range g.editScores {
		scores[file] += score
	}
	return scores, g.totalEditScore + coeditTotal
}

// This scoring function is based on edits.
// Files that are modified by many edits get higher scores.
// Edits that modify many files are weighted less.
// Note this scoring function is independent of the current file.
func (g *graph) computeEditScores() {
	g.editScores = make(map[fileID]float32)
	g.totalEditScore = 0
	for inspect, inspectEdits := range g.files {
		var score float32
		for _, inspectEdit := range inspectEdits {
			score += g.opts.editWeight / float32(g.editSize[inspectEdit])
		}
		g.editScores[inspect] = score
		g.totalEditScore += score
	}
}

// This scoring function is based on co-edits.
// Files that are modified by the same edits as the current file get higher scores.
// Edits that modify many files are weighted less.
func (g graph) computeCoeditScores(base fileID) (map[fileID]float32, float32) {
	baseEditSlice, ok := g.files[base]
	if !ok {
		return nil, 0
	}
	baseEdits := make(map[commitID]struct{})
	for _, baseEdit := range baseEditSlice {
		baseEdits[baseEdit] = struct{}{}
	}

	scores := make(map[fileID]float32)
	var total float32
	for inspect, inspectEdits := range g.files {
		var score float32
		for _, inspectEdit := range inspectEdits {
			if _, ok := baseEdits[inspectEdit]; !ok {
				continue
			}
			score += 1 / float32(g.editSize[inspectEdit])
		}
		scores[inspect] = score
		total += score
	}
	return scores, total
}
