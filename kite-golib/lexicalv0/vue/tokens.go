package vue

var (
	anonSymLt                        = 1
	anonSymGt                        = 2
	anonSymSlashGt                   = 3
	anonSymLtSlash                   = 4
	anonSymEq                        = 5
	symAttributeName                 = 6
	symAttributeValue                = 7
	anonSymSquote                    = 8
	symAttributeValue2               = 9
	anonSymDquote                    = 10
	symAttributeValue3               = 11
	anonSymLbraceLbrace              = 12
	anonSymRbraceRbrace              = 13
	anonSymColon                     = 14
	symDirectiveName                 = 15
	symDirectiveName2                = 16
	auxSymDirectiveArgumentToken1    = 17
	anonSymLbrack                    = 18
	anonSymRbrack                    = 19
	symDirectiveDynamicArgumentValue = 20
	anonSymDot                       = 21
	symTextFragment                  = 22
	symRawText                       = 23
	symTagName                       = 24
	symTagName2                      = 25
	symTagName3                      = 26
	symTagName4                      = 27
	symTagName5                      = 28
	symErroneousEndTagName           = 29
	symImplicitEndTag                = 30
	symRawText2                      = 31
	symComment                       = 32
	symComponent                     = 33
	symNode                          = 34
	symElement                       = 35
	symTemplateElement               = 36
	symScriptElement                 = 37
	symStyleElement                  = 38
	symStartTag                      = 39
	symStartTag2                     = 40
	symStartTag3                     = 41
	symStartTag4                     = 42
	symSelfClosingTag                = 43
	symEndTag                        = 44
	symErroneousEndTag               = 45
	symAttribute                     = 46
	symQuotedAttributeValue          = 47
	symText                          = 48
	symInterpolation                 = 49
	symDirectiveAttribute            = 50
	symDirectiveArgument             = 51
	symDirectiveDynamicArgument      = 52
	symDirectiveModifiers            = 53
	symDirectiveModifier             = 54
	auxSymComponentRepeat1           = 55
	auxSymElementRepeat1             = 56
	auxSymStartTagRepeat1            = 57
	auxSymDirectiveModifiersRepeat1  = 58
)

var allTokens = []int{
	anonSymLt,
	anonSymGt,
	anonSymSlashGt,
	anonSymLtSlash,
	anonSymEq,
	symAttributeName,
	symAttributeValue,
	anonSymSquote,
	symAttributeValue2,
	anonSymDquote,
	symAttributeValue3,
	anonSymLbraceLbrace,
	anonSymRbraceRbrace,
	anonSymColon,
	symDirectiveName,
	symDirectiveName2,
	auxSymDirectiveArgumentToken1,
	anonSymLbrack,
	anonSymRbrack,
	symDirectiveDynamicArgumentValue,
	anonSymDot,
	symTextFragment,
	symRawText,
	symTagName,
	symTagName2,
	symTagName3,
	symTagName4,
	symTagName5,
	symErroneousEndTagName,
	symImplicitEndTag,
	symRawText2,
	symComment,
	symComponent,
	symNode,
	symElement,
	symTemplateElement,
	symScriptElement,
	symStyleElement,
	symStartTag,
	symStartTag2,
	symStartTag3,
	symStartTag4,
	symSelfClosingTag,
	symEndTag,
	symErroneousEndTag,
	symAttribute,
	symQuotedAttributeValue,
	symText,
	symInterpolation,
	symDirectiveAttribute,
	symDirectiveArgument,
	symDirectiveDynamicArgument,
	symDirectiveModifiers,
	symDirectiveModifier,
	auxSymComponentRepeat1,
	auxSymElementRepeat1,
	auxSymStartTagRepeat1,
	auxSymDirectiveModifiersRepeat1,
}
