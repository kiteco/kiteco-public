package html

var (
	anonSymLtBang           = 1
	auxSymDoctypeToken1     = 2
	anonSymGt               = 3
	anonSymDoctype          = 4
	anonSymLt               = 5
	anonSymSlashGt          = 6
	anonSymLtSlash          = 7
	anonSymEq               = 8
	symAttributeName        = 9
	symAttributeValue       = 10
	anonSymSquote           = 11
	symAttributeValue2      = 12
	anonSymDquote           = 13
	symAttributeValue3      = 14
	symText                 = 15
	symTagName              = 16
	symTagName2             = 17
	symTagName3             = 18
	symTagName4             = 19
	symErroneousEndTagName  = 20
	symImplicitEndTag       = 21
	symRawText              = 22
	symComment              = 23
	symFragment             = 24
	symDoctype              = 25
	symNode                 = 26
	symElement              = 27
	symScriptElement        = 28
	symStyleElement         = 29
	symStartTag             = 30
	symStartTag2            = 31
	symStartTag3            = 32
	symSelfClosingTag       = 33
	symEndTag               = 34
	symErroneousEndTag      = 35
	symAttribute            = 36
	symQuotedAttributeValue = 37
	auxSymFragmentRepeat1   = 38
	auxSymStartTagRepeat1   = 39
)

var allTokens = []int{
	anonSymLtBang,
	auxSymDoctypeToken1,
	anonSymGt,
	anonSymDoctype,
	anonSymLt,
	anonSymSlashGt,
	anonSymLtSlash,
	anonSymEq,
	symAttributeName,
	symAttributeValue,
	anonSymSquote,
	symAttributeValue2,
	anonSymDquote,
	symAttributeValue3,
	symText,
	symTagName,
	symTagName2,
	symTagName3,
	symTagName4,
	symErroneousEndTagName,
	symImplicitEndTag,
	symRawText,
	symComment,
	symFragment,
	symDoctype,
	symNode,
	symElement,
	symScriptElement,
	symStyleElement,
	symStartTag,
	symStartTag2,
	symStartTag3,
	symSelfClosingTag,
	symEndTag,
	symErroneousEndTag,
	symAttribute,
	symQuotedAttributeValue,
	auxSymFragmentRepeat1,
	auxSymStartTagRepeat1,
}
