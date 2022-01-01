package golang

const symERROR = 65535

// these token ids are coming from treesitter's internal enum representing
// different token types. for us, this is currently vendored at:
// vendor/github.com/kiteco/go-tree-sitter/golang/parser.c
// w/ the repo @ https://github.com/kiteco/go-tree-sitter
// NOTE: even though these ids start with 1, there is still a token id
// at 0, namely the internal EOF token for treesitter.
var (
	symIdentifier                          = 1
	anonSymLf                              = 2
	anonSymSemi                            = 3
	anonSymPackage                         = 4
	anonSymImport                          = 5
	anonSymDot                             = 6
	symBlankIdentifier                     = 7
	anonSymLparen                          = 8
	anonSymRparen                          = 9
	anonSymConst                           = 10
	anonSymComma                           = 11
	anonSymEq                              = 12
	anonSymVar                             = 13
	anonSymFunc                            = 14
	anonSymDotDotDot                       = 15
	anonSymType                            = 16
	anonSymStar                            = 17
	anonSymLbrack                          = 18
	anonSymRbrack                          = 19
	anonSymStruct                          = 20
	anonSymLbrace                          = 21
	anonSymRbrace                          = 22
	anonSymInterface                       = 23
	anonSymMap                             = 24
	anonSymChan                            = 25
	anonSymLtDash                          = 26
	anonSymColonEq                         = 27
	anonSymPlusPlus                        = 28
	anonSymDashDash                        = 29
	anonSymStarEq                          = 30
	anonSymSlashEq                         = 31
	anonSymPercentEq                       = 32
	anonSymLtLtEq                          = 33
	anonSymGtGtEq                          = 34
	anonSymAmpEq                           = 35
	anonSymAmpCaretEq                      = 36
	anonSymPlusEq                          = 37
	anonSymDashEq                          = 38
	anonSymPipeEq                          = 39
	anonSymCaretEq                         = 40
	anonSymColon                           = 41
	anonSymFallthrough                     = 42
	anonSymBreak                           = 43
	anonSymContinue                        = 44
	anonSymGoto                            = 45
	anonSymReturn                          = 46
	anonSymGo                              = 47
	anonSymDefer                           = 48
	anonSymIf                              = 49
	anonSymElse                            = 50
	anonSymFor                             = 51
	anonSymRange                           = 52
	anonSymSwitch                          = 53
	anonSymCase                            = 54
	anonSymDefault                         = 55
	anonSymSelect                          = 56
	anonSymNew                             = 57
	anonSymMake                            = 58
	anonSymPlus                            = 59
	anonSymDash                            = 60
	anonSymBang                            = 61
	anonSymCaret                           = 62
	anonSymAmp                             = 63
	anonSymSlash                           = 64
	anonSymPercent                         = 65
	anonSymLtLt                            = 66
	anonSymGtGt                            = 67
	anonSymAmpCaret                        = 68
	anonSymPipe                            = 69
	anonSymEqEq                            = 70
	anonSymBangEq                          = 71
	anonSymLt                              = 72
	anonSymLtEq                            = 73
	anonSymGt                              = 74
	anonSymGtEq                            = 75
	anonSymAmpAmp                          = 76
	anonSymPipePipe                        = 77
	symRawStringLiteral                    = 78
	anonSymDquote                          = 79
	auxSymInterpretedStringLiteralToken1   = 80
	symEscapeSequence                      = 81
	symIntLiteral                          = 82
	symFloatLiteral                        = 83
	symImaginaryLiteral                    = 84
	symRuneLiteral                         = 85
	symNil                                 = 86
	symTrue                                = 87
	symFalse                               = 88
	symComment                             = 89
	symSourceFile                          = 90
	symPackageClause                       = 91
	symImportDeclaration                   = 92
	symImportSpec                          = 93
	symDot                                 = 94
	symImportSpecList                      = 95
	symDeclaration                         = 96
	symConstDeclaration                    = 97
	symConstSpec                           = 98
	symVarDeclaration                      = 99
	symVarSpec                             = 100
	symFunctionDeclaration                 = 101
	symMethodDeclaration                   = 102
	symParameterList                       = 103
	symParameterDeclaration                = 104
	symVariadicParameterDeclaration        = 105
	symTypeAlias                           = 106
	symTypeDeclaration                     = 107
	symTypeSpec                            = 108
	symExpressionList                      = 109
	symParenthesizedType                   = 110
	symSimpleType                          = 111
	symPointerType                         = 112
	symArrayType                           = 113
	symImplicitLengthArrayType             = 114
	symSliceType                           = 115
	symStructType                          = 116
	symFieldDeclarationList                = 117
	symFieldDeclaration                    = 118
	symInterfaceType                       = 119
	symMethodSpecList                      = 120
	symMethodSpec                          = 121
	symMapType                             = 122
	symChannelType                         = 123
	symFunctionType                        = 124
	symBlock                               = 125
	symStatementList                       = 126
	symStatement                           = 127
	symEmptyStatement                      = 128
	symSimpleStatement                     = 129
	symSendStatement                       = 130
	symReceiveStatement                    = 131
	symIncStatement                        = 132
	symDecStatement                        = 133
	symAssignmentStatement                 = 134
	symShortVarDeclaration                 = 135
	symLabeledStatement                    = 136
	symEmptyLabeledStatement               = 137
	symFallthroughStatement                = 138
	symBreakStatement                      = 139
	symContinueStatement                   = 140
	symGotoStatement                       = 141
	symReturnStatement                     = 142
	symGoStatement                         = 143
	symDeferStatement                      = 144
	symIfStatement                         = 145
	symForStatement                        = 146
	symForClause                           = 147
	symRangeClause                         = 148
	symExpressionSwitchStatement           = 149
	symExpressionCase                      = 150
	symDefaultCase                         = 151
	symTypeSwitchStatement                 = 152
	symTypeSwitchHeader                    = 153
	symTypeCase                            = 154
	symSelectStatement                     = 155
	symCommunicationCase                   = 156
	symExpression                          = 157
	symParenthesizedExpression             = 158
	symCallExpression                      = 159
	symVariadicArgument                    = 160
	symSpecialArgumentList                 = 161
	symArgumentList                        = 162
	symSelectorExpression                  = 163
	symIndexExpression                     = 164
	symSliceExpression                     = 165
	symTypeAssertionExpression             = 166
	symTypeConversionExpression            = 167
	symCompositeLiteral                    = 168
	symLiteralValue                        = 169
	symKeyedElement                        = 170
	symElement                             = 171
	symFuncLiteral                         = 172
	symUnaryExpression                     = 173
	symBinaryExpression                    = 174
	symQualifiedType                       = 175
	symInterpretedStringLiteral            = 176
	auxSymSourceFileRepeat1                = 177
	auxSymImportSpecListRepeat1            = 178
	auxSymConstDeclarationRepeat1          = 179
	auxSymConstSpecRepeat1                 = 180
	auxSymVarDeclarationRepeat1            = 181
	auxSymParameterListRepeat1             = 182
	auxSymTypeDeclarationRepeat1           = 183
	auxSymFieldNameListRepeat1             = 184
	auxSymExpressionListRepeat1            = 185
	auxSymFieldDeclarationListRepeat1      = 186
	auxSymMethodSpecListRepeat1            = 187
	auxSymStatementListRepeat1             = 188
	auxSymExpressionSwitchStatementRepeat1 = 189
	auxSymTypeSwitchStatementRepeat1       = 190
	auxSymTypeCaseRepeat1                  = 191
	auxSymSelectStatementRepeat1           = 192
	auxSymArgumentListRepeat1              = 193
	auxSymLiteralValueRepeat1              = 194
	auxSymInterpretedStringLiteralRepeat1  = 195
	aliasSymFieldIdentifier                = 196
	aliasSymLabelName                      = 197
	aliasSymPackageIdentifier              = 198
	aliasSymTypeIdentifier                 = 199
)

var _ = []int{
	symIdentifier,
	anonSymLf,
	anonSymSemi,
	anonSymPackage,
	anonSymImport,
	anonSymDot,
	symBlankIdentifier,
	anonSymLparen,
	anonSymRparen,
	anonSymConst,
	anonSymComma,
	anonSymEq,
	anonSymVar,
	anonSymFunc,
	anonSymDotDotDot,
	anonSymType,
	anonSymStar,
	anonSymLbrack,
	anonSymRbrack,
	anonSymStruct,
	anonSymLbrace,
	anonSymRbrace,
	anonSymInterface,
	anonSymMap,
	anonSymChan,
	anonSymLtDash,
	anonSymColonEq,
	anonSymPlusPlus,
	anonSymDashDash,
	anonSymStarEq,
	anonSymSlashEq,
	anonSymPercentEq,
	anonSymLtLtEq,
	anonSymGtGtEq,
	anonSymAmpEq,
	anonSymAmpCaretEq,
	anonSymPlusEq,
	anonSymDashEq,
	anonSymPipeEq,
	anonSymCaretEq,
	anonSymColon,
	anonSymFallthrough,
	anonSymBreak,
	anonSymContinue,
	anonSymGoto,
	anonSymReturn,
	anonSymGo,
	anonSymDefer,
	anonSymIf,
	anonSymElse,
	anonSymFor,
	anonSymRange,
	anonSymSwitch,
	anonSymCase,
	anonSymDefault,
	anonSymSelect,
	anonSymNew,
	anonSymMake,
	anonSymPlus,
	anonSymDash,
	anonSymBang,
	anonSymCaret,
	anonSymAmp,
	anonSymSlash,
	anonSymPercent,
	anonSymLtLt,
	anonSymGtGt,
	anonSymAmpCaret,
	anonSymPipe,
	anonSymEqEq,
	anonSymBangEq,
	anonSymLt,
	anonSymLtEq,
	anonSymGt,
	anonSymGtEq,
	anonSymAmpAmp,
	anonSymPipePipe,
	symRawStringLiteral,
	anonSymDquote,
	auxSymInterpretedStringLiteralToken1,
	symEscapeSequence,
	symIntLiteral,
	symFloatLiteral,
	symImaginaryLiteral,
	symRuneLiteral,
	symNil,
	symTrue,
	symFalse,
	symComment,
	symSourceFile,
	symPackageClause,
	symImportDeclaration,
	symImportSpec,
	symDot,
	symImportSpecList,
	symDeclaration,
	symConstDeclaration,
	symConstSpec,
	symVarDeclaration,
	symVarSpec,
	symFunctionDeclaration,
	symMethodDeclaration,
	symParameterList,
	symParameterDeclaration,
	symVariadicParameterDeclaration,
	symTypeAlias,
	symTypeDeclaration,
	symTypeSpec,
	symExpressionList,
	symParenthesizedType,
	symSimpleType,
	symPointerType,
	symArrayType,
	symImplicitLengthArrayType,
	symSliceType,
	symStructType,
	symFieldDeclarationList,
	symFieldDeclaration,
	symInterfaceType,
	symMethodSpecList,
	symMethodSpec,
	symMapType,
	symChannelType,
	symFunctionType,
	symBlock,
	symStatementList,
	symStatement,
	symEmptyStatement,
	symSimpleStatement,
	symSendStatement,
	symReceiveStatement,
	symIncStatement,
	symDecStatement,
	symAssignmentStatement,
	symShortVarDeclaration,
	symLabeledStatement,
	symEmptyLabeledStatement,
	symFallthroughStatement,
	symBreakStatement,
	symContinueStatement,
	symGotoStatement,
	symReturnStatement,
	symGoStatement,
	symDeferStatement,
	symIfStatement,
	symForStatement,
	symForClause,
	symRangeClause,
	symExpressionSwitchStatement,
	symExpressionCase,
	symDefaultCase,
	symTypeSwitchStatement,
	symTypeSwitchHeader,
	symTypeCase,
	symSelectStatement,
	symCommunicationCase,
	symExpression,
	symParenthesizedExpression,
	symCallExpression,
	symVariadicArgument,
	symSpecialArgumentList,
	symArgumentList,
	symSelectorExpression,
	symIndexExpression,
	symSliceExpression,
	symTypeAssertionExpression,
	symTypeConversionExpression,
	symCompositeLiteral,
	symLiteralValue,
	symKeyedElement,
	symElement,
	symFuncLiteral,
	symUnaryExpression,
	symBinaryExpression,
	symQualifiedType,
	symInterpretedStringLiteral,
	auxSymSourceFileRepeat1,
	auxSymImportSpecListRepeat1,
	auxSymConstDeclarationRepeat1,
	auxSymConstSpecRepeat1,
	auxSymVarDeclarationRepeat1,
	auxSymParameterListRepeat1,
	auxSymTypeDeclarationRepeat1,
	auxSymFieldNameListRepeat1,
	auxSymExpressionListRepeat1,
	auxSymFieldDeclarationListRepeat1,
	auxSymMethodSpecListRepeat1,
	auxSymStatementListRepeat1,
	auxSymExpressionSwitchStatementRepeat1,
	auxSymTypeSwitchStatementRepeat1,
	auxSymTypeCaseRepeat1,
	auxSymSelectStatementRepeat1,
	auxSymArgumentListRepeat1,
	auxSymLiteralValueRepeat1,
	auxSymInterpretedStringLiteralRepeat1,
	aliasSymFieldIdentifier,
	aliasSymLabelName,
	aliasSymPackageIdentifier,
	aliasSymTypeIdentifier,
}
