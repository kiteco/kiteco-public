package stackoverflow

import (
	"sort"
)

type byVotes []*StackOverflowPage

// NetVotes computes the number of upvotes minus the number of downvotes.
func NetVotes(post *StackOverflowPost) int {
	var net int
	for _, v := range post.GetVotes() {
		switch v.GetVoteTypeId() {
		case VoteTypeUpMod:
			net++
		case VoteTypeDownMod:
			net--
		}
	}
	return net
}

func (r byVotes) Less(i, j int) bool {
	// TODO(alex): if we ever need to sort very long lists, then precompute NetVotes.
	// TODO(alex): also use the votes for the answers
	return NetVotes(r[i].GetQuestion()) < NetVotes(r[j].GetQuestion())
}
func (r byVotes) Len() int      { return len(r) }
func (r byVotes) Swap(i, j int) { r[i], r[j] = r[j], r[i] }

// RankInPlace ranks the episode based on the relevance score of an episode for
// an error query and the length of the code that the last event of an episode
// contains. More specifically, score = 0.8 * sim(ep, q) + 0.2 lenScore(ep),
// where lenScore(ep) is proportional to the inverse of the code length in ep,
// and sim(ep, q) is how similar the error message of ep is to q.
func RankInPlace(pages []*StackOverflowPage) {
	sort.Sort(sort.Reverse(byVotes(pages)))
}
