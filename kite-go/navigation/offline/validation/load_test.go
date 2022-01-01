package validation

import (
	"testing"

	"github.com/kiteco/kiteco/kite-go/navigation/recommend"
	"github.com/stretchr/testify/require"
)

type findDiffBlocksTC struct {
	patch         string
	expected      []recommend.Block
	expectedError error
}

func TestFindDiffBlocks(t *testing.T) {
	tcs := []findDiffBlocksTC{
		findDiffBlocksTC{
			patch: `@@ -142,6 +142,7 @@ func (Golang) Provide(ctx kitectx.Context, g Global, in Inputs, out OutputFunc)
 				ModelDurationMS:      int64(modelDuration) / int64(time.Millisecond),
 				CuratedContextExists: len(in.CuratedTokens) > 0,
 				CuratedContextUsed:   in.curatedContextUsed(p.TokenIDs),
+				NumNewlines:          strings.Count(c.Snippet.Text, "\n"),
 			},
 		}`,
			expected: []recommend.Block{
				recommend.Block{
					Content: ` 				ModelDurationMS:      int64(modelDuration) / int64(time.Millisecond),
 				CuratedContextExists: len(in.CuratedTokens) > 0,
 				CuratedContextUsed:   in.curatedContextUsed(p.TokenIDs),
 			},
 		}`,
					FirstLine: 142,
					LastLine:  146,
				},
			},
		},
		findDiffBlocksTC{
			patch: `@@ -40,36 +40,25 @@ func NewCompleteOptions(o data.APIOptions, l lang.Language) CompleteOptions {
 	// per: https://kite.quip.com/1ovKAL1hi2JZ/Spec-Lexical-Completions-Private-Beta#eLcACAqzrwL
 	opts.MixOptions.NestCompletions = false
 
-	var langSupportsNewlines bool
 	switch l {
 	case lang.JavaScript, lang.JSX, lang.Vue, lang.Python:
-		langSupportsNewlines = true
+		opts.MixOptions.AllowCompletionsWithNewlines = true
 	}
 
 	switch o.Editor {
 	case data.VSCodeEditor:
 		opts.MixOptions.PrependCompletionContext = true
-		if langSupportsNewlines {
-			opts.MixOptions.AllowCompletionsWithNewlines = true
-		}
 		opts.MixOptions.NoDollarSignDotCompletions = true
 	case data.SublimeEditor:
 		opts.MixOptions.NoDollarSignCompletions = true
-	case data.AtomEditor:
-		if langSupportsNewlines {
-			opts.MixOptions.AllowCompletionsWithNewlines = true
-		}
+		opts.MixOptions.AllowCompletionsWithNewlines = false
+	case data.VimEditor:
+		opts.MixOptions.AllowCompletionsWithNewlines = false
 	case data.IntelliJEditor:
 		opts.MixOptions.NoSmartStarInHint = true
-		if langSupportsNewlines {
-			opts.MixOptions.AllowCompletionsWithNewlines = true
-		}
 	case data.SpyderEditor:
 		opts.MixOptions.NoSmartStarInHint = true
 		opts.MixOptions.SmartStarInDisplay = true
-		if langSupportsNewlines {
-			opts.MixOptions.AllowCompletionsWithNewlines = true
-		}
 	}
 
`,
			expected: []recommend.Block{
				recommend.Block{
					Content: ` 	// per: https://kite.quip.com/1ovKAL1hi2JZ/Spec-Lexical-Completions-Private-Beta#eLcACAqzrwL
 	opts.MixOptions.NestCompletions = false
 
-	var langSupportsNewlines bool
 	switch l {
 	case lang.JavaScript, lang.JSX, lang.Vue, lang.Python:
-		langSupportsNewlines = true
 	}
 
 	switch o.Editor {
 	case data.VSCodeEditor:
 		opts.MixOptions.PrependCompletionContext = true
-		if langSupportsNewlines {
-			opts.MixOptions.AllowCompletionsWithNewlines = true
-		}
 		opts.MixOptions.NoDollarSignDotCompletions = true
 	case data.SublimeEditor:
 		opts.MixOptions.NoDollarSignCompletions = true
-	case data.AtomEditor:
-		if langSupportsNewlines {
-			opts.MixOptions.AllowCompletionsWithNewlines = true
-		}
 	case data.IntelliJEditor:
 		opts.MixOptions.NoSmartStarInHint = true
-		if langSupportsNewlines {
-			opts.MixOptions.AllowCompletionsWithNewlines = true
-		}
 	case data.SpyderEditor:
 		opts.MixOptions.NoSmartStarInHint = true
 		opts.MixOptions.SmartStarInDisplay = true
-		if langSupportsNewlines {
-			opts.MixOptions.AllowCompletionsWithNewlines = true
-		}
 	}
 
`,
					FirstLine: 40,
					LastLine:  75,
				},
			},
		},
	}
	for _, tc := range tcs {
		blocks, err := findDiffBlocks(tc.patch)
		require.NoError(t, err)
		require.Equal(t, tc.expected, blocks)
	}
}
