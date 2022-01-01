package pythonkeyword

import "github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner"

// Keyword bundles metadata for Python syntactic keywords
type Keyword struct {
	// Cat is the index used for this Keyword in the Keyword model output
	// (to not rely on Keyword order to interpret model result)
	// When cat == -1 the Keyword is not output by the model (ie removed from the training dataset and not present in model outputs)
	Cat int
	// Beginning is true if the Keyword can appear at the Beginning of a line, not including whitespace.
	Beginning bool
	// Middle is true if the Keyword can appear in the Middle of a line (i.e. not at the Beginning)
	Middle bool
	// FollowedBy determines what to add after the Keyword's literal when the completion is chosen.
	FollowedBy string
}

// AllKeywords is a map of all keywords available in python adding some information for each of them
var AllKeywords = map[pythonscanner.Token]Keyword{
	pythonscanner.And:      {Cat: 1, Middle: true, FollowedBy: " "},
	pythonscanner.In:       {Cat: 2, Middle: true, FollowedBy: " "},
	pythonscanner.Is:       {Cat: 3, Middle: true, FollowedBy: " "},
	pythonscanner.Not:      {Cat: 4, Middle: true, FollowedBy: " "},
	pythonscanner.Or:       {Cat: 5, Middle: true, FollowedBy: " "},
	pythonscanner.As:       {Cat: 6, Middle: true, FollowedBy: " "},
	pythonscanner.Assert:   {Cat: 7, Beginning: true, FollowedBy: " "},
	pythonscanner.Break:    {Cat: 8, Beginning: true},
	pythonscanner.Class:    {Cat: 9, Beginning: true, FollowedBy: " "},
	pythonscanner.Continue: {Cat: 10, Beginning: true},
	pythonscanner.Def:      {Cat: 11, Beginning: true, FollowedBy: " "},
	pythonscanner.Del:      {Cat: 12, Beginning: true, FollowedBy: " "},
	pythonscanner.Elif:     {Cat: 13, Beginning: true, FollowedBy: " "},
	pythonscanner.Else:     {Cat: 14, Beginning: true, Middle: true, FollowedBy: ":"},
	pythonscanner.Except:   {Cat: 15, Beginning: true, FollowedBy: " "},
	pythonscanner.Finally:  {Cat: 16, Beginning: true, FollowedBy: ":"},
	pythonscanner.For:      {Cat: 17, Beginning: true, Middle: true, FollowedBy: " "},
	pythonscanner.From:     {Cat: 18, Beginning: true, Middle: true, FollowedBy: " "},
	pythonscanner.If:       {Cat: 19, Beginning: true, Middle: true, FollowedBy: " "},
	pythonscanner.Import:   {Cat: 20, Beginning: true, Middle: true, FollowedBy: " "},
	pythonscanner.Lambda:   {Cat: 21, Beginning: true, Middle: true, FollowedBy: " "},
	pythonscanner.Pass:     {Cat: 22, Beginning: true},
	pythonscanner.Raise:    {Cat: 23, Beginning: true, FollowedBy: " "},
	pythonscanner.Return:   {Cat: 24, Beginning: true, FollowedBy: ""},
	pythonscanner.Try:      {Cat: 25, Beginning: true, FollowedBy: ":"},
	pythonscanner.While:    {Cat: 26, Beginning: true, FollowedBy: " "},
	pythonscanner.With:     {Cat: 27, Beginning: true, FollowedBy: " "},
	pythonscanner.Yield:    {Cat: 28, Beginning: true, FollowedBy: " "},
	pythonscanner.Async:    {Cat: 29, Beginning: true, Middle: true, FollowedBy: " "},
	pythonscanner.Await:    {Cat: 30, Beginning: true, Middle: true, FollowedBy: " "},
	pythonscanner.NonLocal: {Cat: -1, Beginning: true, FollowedBy: " "},
	pythonscanner.Global:   {Cat: -1, Beginning: true, FollowedBy: " "},
}
