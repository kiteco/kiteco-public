package css

var (
	anonSymAtImport                   = 1
	anonSymComma                      = 2
	anonSymSemi                       = 3
	anonSymAtMedia                    = 4
	anonSymAtCharset                  = 5
	anonSymAtNamespace                = 6
	anonSymAtKeyframes                = 7
	symAtKeyword                      = 8
	anonSymLbrace                     = 9
	anonSymRbrace                     = 10
	symFrom                           = 11
	symTo                             = 12
	anonSymAtSupports                 = 13
	symNestingSelector                = 14
	anonSymStar                       = 15
	anonSymDot                        = 16
	anonSymColon                      = 17
	anonSymColonColon                 = 18
	anonSymPound                      = 19
	anonSymLbrack                     = 20
	anonSymEq                         = 21
	anonSymTildeEq                    = 22
	anonSymCaretEq                    = 23
	anonSymPipeEq                     = 24
	anonSymStarEq                     = 25
	anonSymDollarEq                   = 26
	anonSymRbrack                     = 27
	anonSymGt                         = 28
	anonSymTilde                      = 29
	anonSymPlus                       = 30
	anonSymLparen                     = 31
	anonSymRparen                     = 32
	symImportant                      = 33
	anonSymLparen2                    = 34
	anonSymAnd                        = 35
	anonSymOr                         = 36
	anonSymNot                        = 37
	anonSymOnly                       = 38
	anonSymSelector                   = 39
	auxSymColorValueToken1            = 40
	symStringValue                    = 41
	auxSymIntegerValueToken1          = 42
	auxSymFloatValueToken1            = 43
	symUnit                           = 44
	anonSymDash                       = 45
	anonSymBslash                     = 46
	symIdentifier                     = 47
	symAtKeyword2                     = 48
	symComment                        = 49
	symPlainValue                     = 50
	symDescendantOperator             = 51
	symStylesheet                     = 52
	symImportStatement                = 53
	symMediaStatement                 = 54
	symCharsetStatement               = 55
	symNamespaceStatement             = 56
	symKeyframesStatement             = 57
	symKeyframeBlockList              = 58
	symKeyframeBlock                  = 59
	symSupportsStatement              = 60
	symAtRule                         = 61
	symRuleSet                        = 62
	symSelectors                      = 63
	symBlock                          = 64
	symSelector                       = 65
	symUniversalSelector              = 66
	symClassSelector                  = 67
	symPseudoClassSelector            = 68
	symPseudoElementSelector          = 69
	symIDSelector                     = 70
	symAttributeSelector              = 71
	symChildSelector                  = 72
	symDescendantSelector             = 73
	symSiblingSelector                = 74
	symAdjacentSiblingSelector        = 75
	symArguments                      = 76
	symDeclaration                    = 77
	symDeclaration2                   = 78
	symQery                           = 79
	symFeatureQuery                   = 80
	symParenthesizedQuery             = 81
	symBinaryQuery                    = 82
	symUnaryQuery                     = 83
	symSelectorQuery                  = 84
	symValue                          = 85
	symParenthesizedValue             = 86
	symColorValue                     = 87
	symIntegerValue                   = 88
	symFloatValue                     = 89
	symCallExpression                 = 90
	symBinaryExpression               = 91
	symArguments2                     = 92
	auxSymStylesheetRepeat1           = 93
	auxSymImportStatementRepeat1      = 94
	auxSymKeyframeBlockListRepeat1    = 95
	auxSymSelectorsRepeat1            = 96
	auxSymBlockRepeat1                = 97
	auxSymPseudoClassArgumentsRepeat1 = 98
	symAuxPseudoClassArgumentsRepeat2 = 99
	auxSymDeclarationRepeat1          = 100
	auxSymArgumentsRepeat1            = 101
	symAttributeName                  = 102
	symClassName                      = 103
	symFeatureName                    = 104
	symFunctionName                   = 105
	symIDName                         = 106
	symKeyframesName                  = 107
	symKeywordQuery                   = 108
	symNamespaceName                  = 109
	symPropertyName                   = 110
	symTagName                        = 111
)

var allTokens = []int{
	anonSymAtImport,
	anonSymComma,
	anonSymSemi,
	anonSymAtMedia,
	anonSymAtCharset,
	anonSymAtNamespace,
	anonSymAtKeyframes,
	symAtKeyword,
	anonSymLbrace,
	anonSymRbrace,
	symFrom,
	symTo,
	anonSymAtSupports,
	symNestingSelector,
	anonSymStar,
	anonSymDot,
	anonSymColon,
	anonSymColonColon,
	anonSymPound,
	anonSymLbrack,
	anonSymEq,
	anonSymTildeEq,
	anonSymCaretEq,
	anonSymPipeEq,
	anonSymStarEq,
	anonSymDollarEq,
	anonSymRbrack,
	anonSymGt,
	anonSymTilde,
	anonSymPlus,
	anonSymLparen,
	anonSymRparen,
	symImportant,
	anonSymLparen2,
	anonSymAnd,
	anonSymOr,
	anonSymNot,
	anonSymOnly,
	anonSymSelector,
	auxSymColorValueToken1,
	symStringValue,
	auxSymIntegerValueToken1,
	auxSymFloatValueToken1,
	symUnit,
	anonSymDash,
	anonSymBslash,
	symIdentifier,
	symAtKeyword2,
	symComment,
	symPlainValue,
	symDescendantOperator,
	symStylesheet,
	symImportStatement,
	symMediaStatement,
	symCharsetStatement,
	symNamespaceStatement,
	symKeyframesStatement,
	symKeyframeBlockList,
	symKeyframeBlock,
	symSupportsStatement,
	symAtRule,
	symRuleSet,
	symSelectors,
	symBlock,
	symSelector,
	symUniversalSelector,
	symClassSelector,
	symPseudoClassSelector,
	symPseudoElementSelector,
	symIDSelector,
	symAttributeSelector,
	symChildSelector,
	symDescendantSelector,
	symSiblingSelector,
	symAdjacentSiblingSelector,
	symArguments,
	symDeclaration,
	symDeclaration2,
	symQery,
	symFeatureQuery,
	symParenthesizedQuery,
	symBinaryQuery,
	symUnaryQuery,
	symSelectorQuery,
	symValue,
	symParenthesizedValue,
	symColorValue,
	symIntegerValue,
	symFloatValue,
	symCallExpression,
	symBinaryExpression,
	symArguments2,
	auxSymStylesheetRepeat1,
	auxSymImportStatementRepeat1,
	auxSymKeyframeBlockListRepeat1,
	auxSymSelectorsRepeat1,
	auxSymBlockRepeat1,
	auxSymPseudoClassArgumentsRepeat1,
	symAuxPseudoClassArgumentsRepeat2,
	auxSymDeclarationRepeat1,
	auxSymArgumentsRepeat1,
	symAttributeName,
	symClassName,
	symFeatureName,
	symFunctionName,
	symIDName,
	symKeyframesName,
	symKeywordQuery,
	symNamespaceName,
	symPropertyName,
	symTagName,
}
