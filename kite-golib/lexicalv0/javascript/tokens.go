package javascript

var (
	// SymAutomaticSemicolon ...
	SymAutomaticSemicolon = symAutomaticSemicolon
	// SymJsxText ...
	SymJsxText = symJsxText
	// SymComment ...
	SymComment = symComment
)

const symERROR = 65535

// these token ids are coming from treesitter's internal enum representing
// different token types. for us, this is currently vendored at:
// vendor/github.com/kiteco/go-tree-sitter/javascript/parser.c
// w/ the repo @ https://github.com/kiteco/go-tree-sitter
// NOTE: even though these ids start with 1, there is still a token id
// at 0, namely the internal EOF token for treesitter.
var (
	symIdentifier                       = 1
	symHashBangLine                     = 2
	anonSymExport                       = 3
	anonSymStar                         = 4
	anonSymDefault                      = 5
	anonSymLbrace                       = 6
	anonSymComma                        = 7
	anonSymRbrace                       = 8
	anonSymAs                           = 9
	anonSymImport                       = 10
	anonSymFrom                         = 11
	anonSymVar                          = 12
	anonSymLet                          = 13
	anonSymConst                        = 14
	anonSymIf                           = 15
	anonSymElse                         = 16
	anonSymSwitch                       = 17
	anonSymLparen                       = 18
	anonSymRparen                       = 19
	anonSymFor                          = 20
	anonSymIn                           = 21
	anonSymAwait                        = 22
	anonSymOf                           = 23
	anonSymWhile                        = 24
	anonSymDo                           = 25
	anonSymTry                          = 26
	anonSymWith                         = 27
	anonSymBreak                        = 28
	anonSymContinue                     = 29
	anonSymDebugger                     = 30
	anonSymReturn                       = 31
	anonSymThrow                        = 32
	anonSymSemi                         = 33
	anonSymColon                        = 34
	anonSymCase                         = 35
	anonSymCatch                        = 36
	anonSymFinally                      = 37
	anonSymYield                        = 38
	anonSymEq                           = 39
	anonSymLbrack                       = 40
	anonSymRbrack                       = 41
	anonSymLt                           = 42
	anonSymGt                           = 43
	anonSymSlash                        = 44
	symJsxText                          = 45
	symJsxIdentifier                    = 46
	anonSymDot                          = 47
	anonSymClass                        = 48
	anonSymExtends                      = 49
	anonSymAsync                        = 50
	anonSymFunction                     = 51
	anonSymEqGt                         = 52
	anonSymNew                          = 53
	anonSymPlusEq                       = 54
	anonSymDashEq                       = 55
	anonSymStarEq                       = 56
	anonSymSlashEq                      = 57
	anonSymPercentEq                    = 58
	anonSymCaretEq                      = 59
	anonSymAmpEq                        = 60
	anonSymPipeEq                       = 61
	anonSymGtGtEq                       = 62
	anonSymGtGtGtEq                     = 63
	anonSymLtLtEq                       = 64
	anonSymStarStarEq                   = 65
	anonSymDotDotDot                    = 66
	anonSymQmark                        = 67
	anonSymAmpAmp                       = 68
	anonSymPipePipe                     = 69
	anonSymGtGt                         = 70
	anonSymGtGtGt                       = 71
	anonSymLtLt                         = 72
	anonSymAmp                          = 73
	anonSymCaret                        = 74
	anonSymPipe                         = 75
	anonSymPlus                         = 76
	anonSymDash                         = 77
	anonSymPercent                      = 78
	anonSymStarStar                     = 79
	anonSymLtEq                         = 80
	anonSymEqEq                         = 81
	anonSymEqEqEq                       = 82
	anonSymBangEq                       = 83
	anonSymBangEqEq                     = 84
	anonSymGtEq                         = 85
	anonSymInstanceof                   = 86
	anonSymBang                         = 87
	anonSymTilde                        = 88
	anonSymTypeof                       = 89
	anonSymVoid                         = 90
	anonSymDelete                       = 91
	anonSymPlusPlus                     = 92
	anonSymDashDash                     = 93
	anonSymDquote                       = 94
	auxSymStringToken1                  = 95
	anonSymSquote                       = 96
	auxSymStringToken2                  = 97
	symEscapeSequence                   = 98
	symComment                          = 99
	anonSymBquote                       = 100
	anonSymDollarLbrace                 = 101
	anonSymSlash2                       = 102
	symRegexPattern                     = 103
	symRegexFlags                       = 104
	symNumber                           = 105
	anonSymTarget                       = 106
	symThis                             = 107
	symSuper                            = 108
	symTrue                             = 109
	symFalse                            = 110
	symNull                             = 111
	symUndefined                        = 112
	anonSymAt                           = 113
	anonSymStatic                       = 114
	anonSymGet                          = 115
	anonSymSet                          = 116
	symAutomaticSemicolon               = 117
	symTemplateChars                    = 118
	symProgram                          = 119
	symExportStatement                  = 120
	symExportClause                     = 121
	symImportExportSpecifier            = 122
	symDeclaration                      = 123
	symImportStatement                  = 124
	symImportClause                     = 125
	symFromClause                       = 126
	symNamespaceImport                  = 127
	symNamedImports                     = 128
	symExpressionStatement              = 129
	symVariableDeclaration              = 130
	symLexicalDeclaration               = 131
	symVariableDeclarator               = 132
	symStatementBlock                   = 133
	symIfStatement                      = 134
	symSwitchStatement                  = 135
	symForStatement                     = 136
	symForInStatement                   = 137
	symForOfStatement                   = 138
	symWhileStatement                   = 139
	symDoStatement                      = 140
	symTryStatement                     = 141
	symWithStatement                    = 142
	symBreakStatement                   = 143
	symContinueStatement                = 144
	symDebuggerStatement                = 145
	symReturnStatement                  = 146
	symThrowStatement                   = 147
	symEmptyStatement                   = 148
	symLabeledStatement                 = 149
	symSwitchBody                       = 150
	symSwitchCase                       = 151
	symSwitchDefault                    = 152
	symCatchClause                      = 153
	symFinallyClause                    = 154
	symParenthesizedExpression          = 155
	symExpression                       = 156
	symYieldExpression                  = 157
	symObject                           = 158
	symAssignmentPattern                = 159
	symArray                            = 160
	symJsxElement                       = 161
	symJsxFragment                      = 162
	symJsxExpression                    = 163
	symJsxOpeningElement                = 164
	symNestedIdentifier                 = 165
	symJsxNamespaceName                 = 166
	symJsxClosingElement                = 167
	symJsxSelfClosingElement            = 168
	symJsxAttribute                     = 169
	symClass                            = 170
	symClassDeclaration                 = 171
	symClassHeritage                    = 172
	symFunction                         = 173
	symFunctionDeclaration              = 174
	symGeneratorFunction                = 175
	symGeneratorFunctionDeclaration     = 176
	symArrowFunction                    = 177
	symCallExpression                   = 178
	symNewExpression                    = 179
	symAwaitExpression                  = 180
	symMemberExpression                 = 181
	symSubscriptExpression              = 182
	symAssignmentExpression             = 183
	symAugmentedAssignmentExpression    = 184
	symInitializer                      = 185
	symSpreadElement                    = 186
	symTernaryExpression                = 187
	symBinaryExpression                 = 188
	symUnaryExpression                  = 189
	symUpdateExpression                 = 190
	symSequenceExpression               = 191
	symString                           = 192
	symTemplateString                   = 193
	symTemplateSubstitution             = 194
	symRegex                            = 195
	symMetaProperty                     = 196
	symArguments                        = 197
	symDecorator                        = 198
	symDecoratorMemberExpression        = 199
	symDecoratorCallExpression          = 200
	symClassBody                        = 201
	symPublicFieldDefinition            = 202
	symFormalParameters                 = 203
	symRestParameter                    = 204
	symMethodDefinition                 = 205
	symPair                             = 206
	symPropertyName                     = 207
	symComputedPropertyName             = 208
	auxSymProgramRepeat1                = 209
	auxSymExportStatementRepeat1        = 210
	auxSymExportClauseRepeat1           = 211
	auxSymNamedImportsRepeat1           = 212
	auxSymVariableDeclarationRepeat1    = 213
	auxSymSwitchBodyRepeat1             = 214
	auxSymObjectRepeat1                 = 215
	auxSymArrayRepeat1                  = 216
	auxSymJsxElementRepeat1             = 217
	auxSymJsxOpeningElementRepeat1      = 218
	auxSymStringRepeat1                 = 219
	auxSymStringRepeat2                 = 220
	auxSymTemplateStringRepeat1         = 221
	auxSymClassBodyRepeat1              = 222
	auxSymFormalParametersRepeat1       = 223
	aliasSymImportSpecifier             = 224
	aliasSymShorthandPropertyIdentifier = 225
	aliasSymArrayPattern                = 226
	aliasSymPropertyIdentifier          = 227
	aliasSymObjectPattern               = 228
	aliasSymExportSpecifier             = 229
	aliasSymStatementIdentifier         = 230
)

var allTokens = []int{
	symIdentifier,
	symHashBangLine,
	anonSymExport,
	anonSymStar,
	anonSymDefault,
	anonSymLbrace,
	anonSymComma,
	anonSymRbrace,
	anonSymAs,
	anonSymImport,
	anonSymFrom,
	anonSymVar,
	anonSymLet,
	anonSymConst,
	anonSymIf,
	anonSymElse,
	anonSymSwitch,
	anonSymLparen,
	anonSymRparen,
	anonSymFor,
	anonSymIn,
	anonSymAwait,
	anonSymOf,
	anonSymWhile,
	anonSymDo,
	anonSymTry,
	anonSymWith,
	anonSymBreak,
	anonSymContinue,
	anonSymDebugger,
	anonSymReturn,
	anonSymThrow,
	anonSymSemi,
	anonSymColon,
	anonSymCase,
	anonSymCatch,
	anonSymFinally,
	anonSymYield,
	anonSymEq,
	anonSymLbrack,
	anonSymRbrack,
	anonSymLt,
	anonSymGt,
	anonSymSlash,
	symJsxText,
	symJsxIdentifier,
	anonSymDot,
	anonSymClass,
	anonSymExtends,
	anonSymAsync,
	anonSymFunction,
	anonSymEqGt,
	anonSymNew,
	anonSymPlusEq,
	anonSymDashEq,
	anonSymStarEq,
	anonSymSlashEq,
	anonSymPercentEq,
	anonSymCaretEq,
	anonSymAmpEq,
	anonSymPipeEq,
	anonSymGtGtEq,
	anonSymGtGtGtEq,
	anonSymLtLtEq,
	anonSymStarStarEq,
	anonSymDotDotDot,
	anonSymQmark,
	anonSymAmpAmp,
	anonSymPipePipe,
	anonSymGtGt,
	anonSymGtGtGt,
	anonSymLtLt,
	anonSymAmp,
	anonSymCaret,
	anonSymPipe,
	anonSymPlus,
	anonSymDash,
	anonSymPercent,
	anonSymStarStar,
	anonSymLtEq,
	anonSymEqEq,
	anonSymEqEqEq,
	anonSymBangEq,
	anonSymBangEqEq,
	anonSymGtEq,
	anonSymInstanceof,
	anonSymBang,
	anonSymTilde,
	anonSymTypeof,
	anonSymVoid,
	anonSymDelete,
	anonSymPlusPlus,
	anonSymDashDash,
	anonSymDquote,
	auxSymStringToken1,
	anonSymSquote,
	auxSymStringToken2,
	symEscapeSequence,
	symComment,
	anonSymBquote,
	anonSymDollarLbrace,
	anonSymSlash2,
	symRegexPattern,
	symRegexFlags,
	symNumber,
	anonSymTarget,
	symThis,
	symSuper,
	symTrue,
	symFalse,
	symNull,
	symUndefined,
	anonSymAt,
	anonSymStatic,
	anonSymGet,
	anonSymSet,
	symAutomaticSemicolon,
	symTemplateChars,
	symProgram,
	symExportStatement,
	symExportClause,
	symImportExportSpecifier,
	symDeclaration,
	symImportStatement,
	symImportClause,
	symFromClause,
	symNamespaceImport,
	symNamedImports,
	symExpressionStatement,
	symVariableDeclaration,
	symLexicalDeclaration,
	symVariableDeclarator,
	symStatementBlock,
	symIfStatement,
	symSwitchStatement,
	symForStatement,
	symForInStatement,
	symForOfStatement,
	symWhileStatement,
	symDoStatement,
	symTryStatement,
	symWithStatement,
	symBreakStatement,
	symContinueStatement,
	symDebuggerStatement,
	symReturnStatement,
	symThrowStatement,
	symEmptyStatement,
	symLabeledStatement,
	symSwitchBody,
	symSwitchCase,
	symSwitchDefault,
	symCatchClause,
	symFinallyClause,
	symParenthesizedExpression,
	symExpression,
	symYieldExpression,
	symObject,
	symAssignmentPattern,
	symArray,
	symJsxElement,
	symJsxFragment,
	symJsxExpression,
	symJsxOpeningElement,
	symNestedIdentifier,
	symJsxNamespaceName,
	symJsxClosingElement,
	symJsxSelfClosingElement,
	symJsxAttribute,
	symClass,
	symClassDeclaration,
	symClassHeritage,
	symFunction,
	symFunctionDeclaration,
	symGeneratorFunction,
	symGeneratorFunctionDeclaration,
	symArrowFunction,
	symCallExpression,
	symNewExpression,
	symAwaitExpression,
	symMemberExpression,
	symSubscriptExpression,
	symAssignmentExpression,
	symAugmentedAssignmentExpression,
	symInitializer,
	symSpreadElement,
	symTernaryExpression,
	symBinaryExpression,
	symUnaryExpression,
	symUpdateExpression,
	symSequenceExpression,
	symString,
	symTemplateString,
	symTemplateSubstitution,
	symRegex,
	symMetaProperty,
	symArguments,
	symDecorator,
	symDecoratorMemberExpression,
	symDecoratorCallExpression,
	symClassBody,
	symPublicFieldDefinition,
	symFormalParameters,
	symRestParameter,
	symMethodDefinition,
	symPair,
	symPropertyName,
	symComputedPropertyName,
	auxSymProgramRepeat1,
	auxSymExportStatementRepeat1,
	auxSymExportClauseRepeat1,
	auxSymNamedImportsRepeat1,
	auxSymVariableDeclarationRepeat1,
	auxSymSwitchBodyRepeat1,
	auxSymObjectRepeat1,
	auxSymArrayRepeat1,
	auxSymJsxElementRepeat1,
	auxSymJsxOpeningElementRepeat1,
	auxSymStringRepeat1,
	auxSymStringRepeat2,
	auxSymTemplateStringRepeat1,
	auxSymClassBodyRepeat1,
	auxSymFormalParametersRepeat1,
	aliasSymImportSpecifier,
	aliasSymShorthandPropertyIdentifier,
	aliasSymArrayPattern,
	aliasSymPropertyIdentifier,
	aliasSymObjectPattern,
	aliasSymExportSpecifier,
	aliasSymStatementIdentifier,
}
