package python

var (
	// SymString ...
	SymString = symString
	// SymComment ...
	SymComment = symComment
	// AnonSymDquoteStart ...
	AnonSymDquoteStart = anonSymDquoteStart
	// AnonSymDquoteEnd ...
	AnonSymDquoteEnd = anonSymDquoteEnd
)

var (
	symIdentifier                     = 1
	anonSymImport                     = 2
	anonSymDot                        = 3
	anonSymFrom                       = 4
	anonSymFuture                     = 5
	anonSymLparen                     = 6
	anonSymRparen                     = 7
	anonSymComma                      = 8
	anonSymAs                         = 9
	anonSymStar                       = 10
	anonSymPrint                      = 11
	anonSymGtGt                       = 12
	anonSymAssert                     = 13
	anonSymColonEq                    = 14
	anonSymReturn                     = 15
	anonSymDel                        = 16
	anonSymRaise                      = 17
	anonSymPass                       = 18
	anonSymBreak                      = 19
	anonSymContinue                   = 20
	anonSymIf                         = 21
	anonSymColon                      = 22
	anonSymElif                       = 23
	anonSymElse                       = 24
	anonSymAsync                      = 25
	anonSymFor                        = 26
	anonSymIn                         = 27
	anonSymWhile                      = 28
	anonSymTry                        = 29
	anonSymExcept                     = 30
	anonSymFinally                    = 31
	anonSymWith                       = 32
	anonSymDef                        = 33
	anonSymRarrow                     = 34
	anonSymEq                         = 35
	anonSymStarStar                   = 36
	anonSymGlobal                     = 37
	anonSymNonlocal                   = 38
	anonSymExec                       = 39
	anonSymClass                      = 40
	anonSymAt                         = 41
	anonSymNot                        = 42
	anonSymAnd                        = 43
	anonSymOr                         = 44
	anonSymPlus                       = 45
	anonSymDash                       = 46
	anonSymSlash                      = 47
	anonSymPercent                    = 48
	anonSymSlashSlash                 = 49
	anonSymPipe                       = 50
	anonSymAmp                        = 51
	anonSymCaret                      = 52
	anonSymLtLt                       = 53
	anonSymTilde                      = 54
	anonSymLt                         = 55
	anonSymLtEq                       = 56
	anonSymEqEq                       = 57
	anonSymBangEq                     = 58
	anonSymGtEq                       = 59
	anonSymGt                         = 60
	anonSymLtGt                       = 61
	anonSymIs                         = 62
	anonSymLambda                     = 63
	anonSymPlusEq                     = 64
	anonSymDashEq                     = 65
	anonSymStarEq                     = 66
	anonSymSlashEq                    = 67
	anonSymAtEq                       = 68
	anonSymSlashSlashEq               = 69
	anonSymPercentEq                  = 70
	anonSymStarStarEq                 = 71
	anonSymGtGtEq                     = 72
	anonSymLtLtEq                     = 73
	anonSymAmpEq                      = 74
	anonSymCaretEq                    = 75
	anonSymPipeEq                     = 76
	anonSymYield                      = 77
	anonSymLbrack                     = 78
	anonSymRbrack                     = 79
	symEllipsis                       = 80
	anonSymLbrace                     = 81
	anonSymRbrace                     = 82
	symEscapeSequence                 = 83
	symNotEscapeSequence              = 84
	auxSymFormatSpecifierToken1       = 85
	symTypeConversion                 = 86
	symInteger                        = 87
	symFloat                          = 88
	symTrue                           = 89
	symFalse                          = 90
	symNone                           = 91
	anonSymAwait                      = 92
	symComment                        = 93
	symSemicolon                      = 94
	symNewline                        = 95
	symIndent                         = 96
	symDedent                         = 97
	anonSymDquoteStart                = 98 // ", the start quote
	symStringContent                  = 99
	anonSymDquoteEnd                  = 100 // also ", but the end quote
	symModule                         = 101
	symStatement                      = 102
	symSimpleStatements               = 103
	symImportStatement                = 104
	symImportPrefix                   = 105
	symRelativeImport                 = 106
	symFutureImportStatement          = 107
	symImportFromStatement            = 108
	symImportList                     = 109
	symAliasedImport                  = 110
	symWildcardImport                 = 111
	symPrintStatement                 = 112
	symChevron                        = 113
	symAssertStatement                = 114
	symExpressionStatement            = 115
	symNamedExpression                = 116
	symReturnStatement                = 117
	symDeleteStatement                = 118
	symRaiseStatement                 = 119
	symPassStatement                  = 120
	symBreakStatement                 = 121
	symContinueStatement              = 122
	symIfStatement                    = 123
	symElifClause                     = 124
	symElseClause                     = 125
	symForStatement                   = 126
	symWhileStatement                 = 127
	symTryStatement                   = 128
	symExceptClause                   = 129
	symFinallyClause                  = 130
	symWithStatement                  = 131
	symWithItem                       = 132
	symFunctionDefinition             = 133
	symParameters                     = 134
	symLambdaParameters               = 135
	symParameters2                    = 136
	symDefaultParameter               = 137
	symTypedDefaultParameter          = 138
	symListSplat                      = 139
	symDictionarySplat                = 140
	symGlobalStatement                = 141
	symNonlocalStatement              = 142
	symExecStatement                  = 143
	symClassDefinition                = 144
	symParenthesizedExpression        = 145
	symArgumentList                   = 146
	symDecoratedDefinition            = 147
	symDecorator                      = 148
	symBlock                          = 149
	symVariables                      = 150
	symExpressionList                 = 151
	symDottedName                     = 152
	symExpressionWithinForInClause    = 153
	symExpression                     = 154
	symPrimaryExpression              = 155
	symNotOperator                    = 156
	symBooleanOperator                = 157
	symBinaryOperator                 = 158
	symUnaryOperator                  = 159
	symComparisonOperator             = 160
	symLambda                         = 161
	symLambda2                        = 162
	symAssignment                     = 163
	symAugmentedAssignment            = 164
	symRightHandSide                  = 165
	symYield                          = 166
	symAttribute                      = 167
	symSubscript                      = 168
	symSlice                          = 169
	symCall                           = 170
	symTypedParameter                 = 171
	symType                           = 172
	symKeywordArgument                = 173
	symList                           = 174
	symComprehensionClauses           = 175
	symListComprehension              = 176
	symDictionary                     = 177
	symDictionaryComprehension        = 178
	symPair                           = 179
	symSet                            = 180
	symSetComprehension               = 181
	symParenthesizedExpression2       = 182
	symTuple                          = 183
	symGeneratorExpression            = 184
	symForInClause                    = 185
	symIfClause                       = 186
	symConditionalExpression          = 187
	symConcatenatedString             = 188
	symString                         = 189
	symInterpolation                  = 190
	symFormatSpecifier                = 191
	symFormatExpression               = 192
	symAwait                          = 193
	auxSymModuleRepeat1               = 194
	auxSymSimpleStatementsRepeat1     = 195
	auxSymImportPrefixRepeat1         = 196
	auxSymImportListRepeat1           = 197
	auxSymPrintStatementRepeat1       = 198
	auxSymAssertStatementRepeat1      = 199
	auxSymIfStatementRepeat1          = 200
	auxSymTryStatementRepeat1         = 201
	auxSymWithStatementRepeat1        = 202
	auxSymParametersRepeat1           = 203
	auxSymGlobalStatementRepeat1      = 204
	auxSymArgumentListRepeat1         = 205
	auxSymDecoratedDefinitionRepeat1  = 206
	auxSymVariablesRepeat1            = 207
	auxSymDottedNameRepeat1           = 208
	auxSymComparisonOperatorRepeat1   = 209
	auxSymSubscriptRepeat1            = 210
	auxSymListRepeat1                 = 211
	auxSymComprehensionClausesRepeat1 = 212
	auxSymDictionaryRepeat1           = 213
	auxSymTupleRepeat1                = 214
	auxSymForInClauseRepeat1          = 215
	auxSymConcatenatedStringRepeat1   = 216
	auxSymStringRepeat1               = 217
	auxSymFormatSpecifierRepeat1      = 218
	endOfStatement                    = 219
	startOfBlock                      = 220
	endOfBlock                        = 221
)

var allTokens = []int{
	symIdentifier,
	anonSymImport,
	anonSymDot,
	anonSymFrom,
	anonSymFuture,
	anonSymLparen,
	anonSymRparen,
	anonSymComma,
	anonSymAs,
	anonSymStar,
	anonSymPrint,
	anonSymGtGt,
	anonSymAssert,
	anonSymColonEq,
	anonSymReturn,
	anonSymDel,
	anonSymRaise,
	anonSymPass,
	anonSymBreak,
	anonSymContinue,
	anonSymIf,
	anonSymColon,
	anonSymElif,
	anonSymElse,
	anonSymAsync,
	anonSymFor,
	anonSymIn,
	anonSymWhile,
	anonSymTry,
	anonSymExcept,
	anonSymFinally,
	anonSymWith,
	anonSymDef,
	anonSymRarrow,
	anonSymEq,
	anonSymStarStar,
	anonSymGlobal,
	anonSymNonlocal,
	anonSymExec,
	anonSymClass,
	anonSymAt,
	anonSymNot,
	anonSymAnd,
	anonSymOr,
	anonSymPlus,
	anonSymDash,
	anonSymSlash,
	anonSymPercent,
	anonSymSlashSlash,
	anonSymPipe,
	anonSymAmp,
	anonSymCaret,
	anonSymLtLt,
	anonSymTilde,
	anonSymLt,
	anonSymLtEq,
	anonSymEqEq,
	anonSymBangEq,
	anonSymGtEq,
	anonSymGt,
	anonSymLtGt,
	anonSymIs,
	anonSymLambda,
	anonSymPlusEq,
	anonSymDashEq,
	anonSymStarEq,
	anonSymSlashEq,
	anonSymAtEq,
	anonSymSlashSlashEq,
	anonSymPercentEq,
	anonSymStarStarEq,
	anonSymGtGtEq,
	anonSymLtLtEq,
	anonSymAmpEq,
	anonSymCaretEq,
	anonSymPipeEq,
	anonSymYield,
	anonSymLbrack,
	anonSymRbrack,
	symEllipsis,
	anonSymLbrace,
	anonSymRbrace,
	symEscapeSequence,
	symNotEscapeSequence,
	auxSymFormatSpecifierToken1,
	symTypeConversion,
	symInteger,
	symFloat,
	symTrue,
	symFalse,
	symNone,
	anonSymAwait,
	symComment,
	symSemicolon,
	symNewline,
	symIndent,
	symDedent,
	anonSymDquoteStart,
	symStringContent,
	anonSymDquoteEnd,
	symModule,
	symStatement,
	symSimpleStatements,
	symImportStatement,
	symImportPrefix,
	symRelativeImport,
	symFutureImportStatement,
	symImportFromStatement,
	symImportList,
	symAliasedImport,
	symWildcardImport,
	symPrintStatement,
	symChevron,
	symAssertStatement,
	symExpressionStatement,
	symNamedExpression,
	symReturnStatement,
	symDeleteStatement,
	symRaiseStatement,
	symPassStatement,
	symBreakStatement,
	symContinueStatement,
	symIfStatement,
	symElifClause,
	symElseClause,
	symForStatement,
	symWhileStatement,
	symTryStatement,
	symExceptClause,
	symFinallyClause,
	symWithStatement,
	symWithItem,
	symFunctionDefinition,
	symParameters,
	symLambdaParameters,
	symParameters2,
	symDefaultParameter,
	symTypedDefaultParameter,
	symListSplat,
	symDictionarySplat,
	symGlobalStatement,
	symNonlocalStatement,
	symExecStatement,
	symClassDefinition,
	symParenthesizedExpression,
	symArgumentList,
	symDecoratedDefinition,
	symDecorator,
	symBlock,
	symVariables,
	symExpressionList,
	symDottedName,
	symExpressionWithinForInClause,
	symExpression,
	symPrimaryExpression,
	symNotOperator,
	symBooleanOperator,
	symBinaryOperator,
	symUnaryOperator,
	symComparisonOperator,
	symLambda,
	symLambda2,
	symAssignment,
	symAugmentedAssignment,
	symRightHandSide,
	symYield,
	symAttribute,
	symSubscript,
	symSlice,
	symCall,
	symTypedParameter,
	symType,
	symKeywordArgument,
	symList,
	symComprehensionClauses,
	symListComprehension,
	symDictionary,
	symDictionaryComprehension,
	symPair,
	symSet,
	symSetComprehension,
	symParenthesizedExpression2,
	symTuple,
	symGeneratorExpression,
	symForInClause,
	symIfClause,
	symConditionalExpression,
	symConcatenatedString,
	symString,
	symInterpolation,
	symFormatSpecifier,
	symFormatExpression,
	symAwait,
	auxSymModuleRepeat1,
	auxSymSimpleStatementsRepeat1,
	auxSymImportPrefixRepeat1,
	auxSymImportListRepeat1,
	auxSymPrintStatementRepeat1,
	auxSymAssertStatementRepeat1,
	auxSymIfStatementRepeat1,
	auxSymTryStatementRepeat1,
	auxSymWithStatementRepeat1,
	auxSymParametersRepeat1,
	auxSymGlobalStatementRepeat1,
	auxSymArgumentListRepeat1,
	auxSymDecoratedDefinitionRepeat1,
	auxSymVariablesRepeat1,
	auxSymDottedNameRepeat1,
	auxSymComparisonOperatorRepeat1,
	auxSymSubscriptRepeat1,
	auxSymListRepeat1,
	auxSymComprehensionClausesRepeat1,
	auxSymDictionaryRepeat1,
	auxSymTupleRepeat1,
	auxSymForInClauseRepeat1,
	auxSymConcatenatedStringRepeat1,
	auxSymStringRepeat1,
	auxSymFormatSpecifierRepeat1,
	endOfStatement,
	startOfBlock,
	endOfBlock,
}
