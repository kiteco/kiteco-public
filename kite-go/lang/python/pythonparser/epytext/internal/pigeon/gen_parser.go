package pigeon

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"os"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

var g = &grammar{
	rules: []*rule{
		{
			name: "Grammar",
			pos:  position{line: 14, col: 1, offset: 322},
			expr: &actionExpr{
				pos: position{line: 15, col: 3, offset: 335},
				run: (*parser).callonGrammar1,
				expr: &seqExpr{
					pos: position{line: 15, col: 3, offset: 335},
					exprs: []interface{}{
						&stateCodeExpr{
							pos: position{line: 15, col: 3, offset: 335},
							run: (*parser).callonGrammar3,
						},
						&labeledExpr{
							pos:   position{line: 18, col: 3, offset: 368},
							label: "blocks",
							expr: &zeroOrMoreExpr{
								pos: position{line: 18, col: 10, offset: 375},
								expr: &actionExpr{
									pos: position{line: 36, col: 3, offset: 828},
									run: (*parser).callonGrammar6,
									expr: &labeledExpr{
										pos:   position{line: 36, col: 3, offset: 828},
										label: "block",
										expr: &choiceExpr{
											pos: position{line: 37, col: 7, offset: 842},
											alternatives: []interface{}{
												&actionExpr{
													pos: position{line: 37, col: 7, offset: 842},
													run: (*parser).callonGrammar9,
													expr: &seqExpr{
														pos: position{line: 37, col: 7, offset: 842},
														exprs: []interface{}{
															&labeledExpr{
																pos:   position{line: 37, col: 7, offset: 842},
																label: "lit",
																expr: &actionExpr{
																	pos: position{line: 55, col: 3, offset: 1550},
																	run: (*parser).callonGrammar12,
																	expr: &seqExpr{
																		pos: position{line: 55, col: 3, offset: 1550},
																		exprs: []interface{}{
																			&andCodeExpr{
																				pos: position{line: 55, col: 3, offset: 1550},
																				run: (*parser).callonGrammar14,
																			},
																			&labeledExpr{
																				pos:   position{line: 59, col: 3, offset: 1664},
																				label: "lines",
																				expr: &oneOrMoreExpr{
																					pos: position{line: 59, col: 9, offset: 1670},
																					expr: &seqExpr{
																						pos: position{line: 60, col: 5, offset: 1676},
																						exprs: []interface{}{
																							&zeroOrMoreExpr{
																								pos: position{line: 235, col: 6, offset: 6555},
																								expr: &seqExpr{
																									pos: position{line: 234, col: 14, offset: 6534},
																									exprs: []interface{}{
																										&zeroOrMoreExpr{
																											pos: position{line: 234, col: 14, offset: 6534},
																											expr: &charClassMatcher{
																												pos:             position{line: 233, col: 15, offset: 6513},
																												val:             "[ \\t\\f]",
																												chars:           []rune{' ', '\t', '\f'},
																												basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, true, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
																												ignoreCase:      false,
																												inverted:        false,
																											},
																										},
																										&choiceExpr{
																											pos: position{line: 230, col: 10, offset: 6451},
																											alternatives: []interface{}{
																												&litMatcher{
																													pos:        position{line: 230, col: 10, offset: 6451},
																													val:        "\r\n\n\n\n\n\n\n\n\n\n",
																													ignoreCase: false,
																												},
																												&charClassMatcher{
																													pos:             position{line: 230, col: 22, offset: 6463},
																													val:             "[\\r\\n]",
																													chars:           []rune{'\r', '\n'},
																													basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, false, true, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
																													ignoreCase:      false,
																													inverted:        false,
																												},
																											},
																										},
																									},
																								},
																							},
																							&labeledExpr{
																								pos:   position{line: 60, col: 7, offset: 1678},
																								label: "line",
																								expr: &actionExpr{
																									pos: position{line: 213, col: 17, offset: 6111},
																									run: (*parser).callonGrammar26,
																									expr: &seqExpr{
																										pos: position{line: 213, col: 17, offset: 6111},
																										exprs: []interface{}{
																											&labeledExpr{
																												pos:   position{line: 225, col: 11, offset: 6364},
																												label: "indent",
																												expr: &zeroOrMoreExpr{
																													pos: position{line: 225, col: 18, offset: 6371},
																													expr: &charClassMatcher{
																														pos:             position{line: 232, col: 16, offset: 6493},
																														val:             "[ \\t]",
																														chars:           []rune{' ', '\t'},
																														basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
																														ignoreCase:      false,
																														inverted:        false,
																													},
																												},
																											},
																											&stateCodeExpr{
																												pos: position{line: 226, col: 3, offset: 6386},
																												run: (*parser).callonGrammar31,
																											},
																											&labeledExpr{
																												pos:   position{line: 213, col: 24, offset: 6118},
																												label: "text",
																												expr: &oneOrMoreExpr{
																													pos: position{line: 213, col: 29, offset: 6123},
																													expr: &seqExpr{
																														pos: position{line: 213, col: 31, offset: 6125},
																														exprs: []interface{}{
																															&notExpr{
																																pos: position{line: 213, col: 31, offset: 6125},
																																expr: &choiceExpr{
																																	pos: position{line: 230, col: 10, offset: 6451},
																																	alternatives: []interface{}{
																																		&litMatcher{
																																			pos:        position{line: 230, col: 10, offset: 6451},
																																			val:        "\r\n\n\n\n\n\n\n\n\n\n",
																																			ignoreCase: false,
																																		},
																																		&charClassMatcher{
																																			pos:             position{line: 230, col: 22, offset: 6463},
																																			val:             "[\\r\\n]",
																																			chars:           []rune{'\r', '\n'},
																																			basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, false, true, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
																																			ignoreCase:      false,
																																			inverted:        false,
																																		},
																																	},
																																},
																															},
																															&anyMatcher{
																																line: 213, col: 36, offset: 6130,
																															},
																														},
																													},
																												},
																											},
																											&choiceExpr{
																												pos: position{line: 230, col: 10, offset: 6451},
																												alternatives: []interface{}{
																													&litMatcher{
																														pos:        position{line: 230, col: 10, offset: 6451},
																														val:        "\r\n\n\n\n\n\n\n\n\n\n",
																														ignoreCase: false,
																													},
																													&charClassMatcher{
																														pos:             position{line: 230, col: 22, offset: 6463},
																														val:             "[\\r\\n]",
																														chars:           []rune{'\r', '\n'},
																														basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, false, true, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
																														ignoreCase:      false,
																														inverted:        false,
																													},
																												},
																											},
																										},
																									},
																								},
																							},
																							&andCodeExpr{
																								pos: position{line: 61, col: 5, offset: 1700},
																								run: (*parser).callonGrammar43,
																							},
																						},
																					},
																				},
																			},
																		},
																	},
																},
															},
															&stateCodeExpr{
																pos: position{line: 37, col: 19, offset: 854},
																run: (*parser).callonGrammar44,
															},
														},
													},
												},
												&actionExpr{
													pos: position{line: 75, col: 3, offset: 2013},
													run: (*parser).callonGrammar45,
													expr: &seqExpr{
														pos: position{line: 75, col: 3, offset: 2013},
														exprs: []interface{}{
															&zeroOrMoreExpr{
																pos: position{line: 235, col: 6, offset: 6555},
																expr: &seqExpr{
																	pos: position{line: 234, col: 14, offset: 6534},
																	exprs: []interface{}{
																		&zeroOrMoreExpr{
																			pos: position{line: 234, col: 14, offset: 6534},
																			expr: &charClassMatcher{
																				pos:             position{line: 233, col: 15, offset: 6513},
																				val:             "[ \\t\\f]",
																				chars:           []rune{' ', '\t', '\f'},
																				basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, true, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
																				ignoreCase:      false,
																				inverted:        false,
																			},
																		},
																		&choiceExpr{
																			pos: position{line: 230, col: 10, offset: 6451},
																			alternatives: []interface{}{
																				&litMatcher{
																					pos:        position{line: 230, col: 10, offset: 6451},
																					val:        "\r\n\n\n\n\n\n\n\n\n\n",
																					ignoreCase: false,
																				},
																				&charClassMatcher{
																					pos:             position{line: 230, col: 22, offset: 6463},
																					val:             "[\\r\\n]",
																					chars:           []rune{'\r', '\n'},
																					basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, false, true, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
																					ignoreCase:      false,
																					inverted:        false,
																				},
																			},
																		},
																	},
																},
															},
															&labeledExpr{
																pos:   position{line: 75, col: 5, offset: 2015},
																label: "header",
																expr: &actionExpr{
																	pos: position{line: 213, col: 17, offset: 6111},
																	run: (*parser).callonGrammar55,
																	expr: &seqExpr{
																		pos: position{line: 213, col: 17, offset: 6111},
																		exprs: []interface{}{
																			&labeledExpr{
																				pos:   position{line: 225, col: 11, offset: 6364},
																				label: "indent",
																				expr: &zeroOrMoreExpr{
																					pos: position{line: 225, col: 18, offset: 6371},
																					expr: &charClassMatcher{
																						pos:             position{line: 232, col: 16, offset: 6493},
																						val:             "[ \\t]",
																						chars:           []rune{' ', '\t'},
																						basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
																						ignoreCase:      false,
																						inverted:        false,
																					},
																				},
																			},
																			&stateCodeExpr{
																				pos: position{line: 226, col: 3, offset: 6386},
																				run: (*parser).callonGrammar60,
																			},
																			&labeledExpr{
																				pos:   position{line: 213, col: 24, offset: 6118},
																				label: "text",
																				expr: &oneOrMoreExpr{
																					pos: position{line: 213, col: 29, offset: 6123},
																					expr: &seqExpr{
																						pos: position{line: 213, col: 31, offset: 6125},
																						exprs: []interface{}{
																							&notExpr{
																								pos: position{line: 213, col: 31, offset: 6125},
																								expr: &choiceExpr{
																									pos: position{line: 230, col: 10, offset: 6451},
																									alternatives: []interface{}{
																										&litMatcher{
																											pos:        position{line: 230, col: 10, offset: 6451},
																											val:        "\r\n\n\n\n\n\n\n\n\n\n",
																											ignoreCase: false,
																										},
																										&charClassMatcher{
																											pos:             position{line: 230, col: 22, offset: 6463},
																											val:             "[\\r\\n]",
																											chars:           []rune{'\r', '\n'},
																											basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, false, true, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
																											ignoreCase:      false,
																											inverted:        false,
																										},
																									},
																								},
																							},
																							&anyMatcher{
																								line: 213, col: 36, offset: 6130,
																							},
																						},
																					},
																				},
																			},
																			&choiceExpr{
																				pos: position{line: 230, col: 10, offset: 6451},
																				alternatives: []interface{}{
																					&litMatcher{
																						pos:        position{line: 230, col: 10, offset: 6451},
																						val:        "\r\n\n\n\n\n\n\n\n\n\n",
																						ignoreCase: false,
																					},
																					&charClassMatcher{
																						pos:             position{line: 230, col: 22, offset: 6463},
																						val:             "[\\r\\n]",
																						chars:           []rune{'\r', '\n'},
																						basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, false, true, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
																						ignoreCase:      false,
																						inverted:        false,
																					},
																				},
																			},
																		},
																	},
																},
															},
															&labeledExpr{
																pos:   position{line: 75, col: 25, offset: 2035},
																label: "underline",
																expr: &actionExpr{
																	pos: position{line: 83, col: 18, offset: 2244},
																	run: (*parser).callonGrammar73,
																	expr: &seqExpr{
																		pos: position{line: 83, col: 18, offset: 2244},
																		exprs: []interface{}{
																			&labeledExpr{
																				pos:   position{line: 225, col: 11, offset: 6364},
																				label: "indent",
																				expr: &zeroOrMoreExpr{
																					pos: position{line: 225, col: 18, offset: 6371},
																					expr: &charClassMatcher{
																						pos:             position{line: 232, col: 16, offset: 6493},
																						val:             "[ \\t]",
																						chars:           []rune{' ', '\t'},
																						basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
																						ignoreCase:      false,
																						inverted:        false,
																					},
																				},
																			},
																			&stateCodeExpr{
																				pos: position{line: 226, col: 3, offset: 6386},
																				run: (*parser).callonGrammar78,
																			},
																			&labeledExpr{
																				pos:   position{line: 83, col: 25, offset: 2251},
																				label: "underline",
																				expr: &actionExpr{
																					pos: position{line: 88, col: 21, offset: 2337},
																					run: (*parser).callonGrammar80,
																					expr: &choiceExpr{
																						pos: position{line: 88, col: 23, offset: 2339},
																						alternatives: []interface{}{
																							&oneOrMoreExpr{
																								pos: position{line: 88, col: 23, offset: 2339},
																								expr: &litMatcher{
																									pos:        position{line: 88, col: 23, offset: 2339},
																									val:        "=",
																									ignoreCase: false,
																								},
																							},
																							&oneOrMoreExpr{
																								pos: position{line: 88, col: 30, offset: 2346},
																								expr: &litMatcher{
																									pos:        position{line: 88, col: 30, offset: 2346},
																									val:        "-",
																									ignoreCase: false,
																								},
																							},
																							&oneOrMoreExpr{
																								pos: position{line: 88, col: 37, offset: 2353},
																								expr: &litMatcher{
																									pos:        position{line: 88, col: 37, offset: 2353},
																									val:        "~",
																									ignoreCase: false,
																								},
																							},
																						},
																					},
																				},
																			},
																			&choiceExpr{
																				pos: position{line: 230, col: 10, offset: 6451},
																				alternatives: []interface{}{
																					&litMatcher{
																						pos:        position{line: 230, col: 10, offset: 6451},
																						val:        "\r\n\n\n\n\n\n\n\n\n\n",
																						ignoreCase: false,
																					},
																					&charClassMatcher{
																						pos:             position{line: 230, col: 22, offset: 6463},
																						val:             "[\\r\\n]",
																						chars:           []rune{'\r', '\n'},
																						basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, false, true, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
																						ignoreCase:      false,
																						inverted:        false,
																					},
																				},
																			},
																		},
																	},
																},
															},
															&andCodeExpr{
																pos: position{line: 76, col: 3, offset: 2061},
																run: (*parser).callonGrammar91,
															},
														},
													},
												},
												&actionExpr{
													pos: position{line: 39, col: 7, offset: 941},
													run: (*parser).callonGrammar92,
													expr: &seqExpr{
														pos: position{line: 39, col: 7, offset: 941},
														exprs: []interface{}{
															&labeledExpr{
																pos:   position{line: 39, col: 7, offset: 941},
																label: "doc",
																expr: &actionExpr{
																	pos: position{line: 185, col: 12, offset: 5425},
																	run: (*parser).callonGrammar95,
																	expr: &seqExpr{
																		pos: position{line: 185, col: 12, offset: 5425},
																		exprs: []interface{}{
																			&oneOrMoreExpr{
																				pos: position{line: 185, col: 12, offset: 5425},
																				expr: &seqExpr{
																					pos: position{line: 234, col: 14, offset: 6534},
																					exprs: []interface{}{
																						&zeroOrMoreExpr{
																							pos: position{line: 234, col: 14, offset: 6534},
																							expr: &charClassMatcher{
																								pos:             position{line: 233, col: 15, offset: 6513},
																								val:             "[ \\t\\f]",
																								chars:           []rune{' ', '\t', '\f'},
																								basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, true, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
																								ignoreCase:      false,
																								inverted:        false,
																							},
																						},
																						&choiceExpr{
																							pos: position{line: 230, col: 10, offset: 6451},
																							alternatives: []interface{}{
																								&litMatcher{
																									pos:        position{line: 230, col: 10, offset: 6451},
																									val:        "\r\n\n\n\n\n\n\n\n\n\n",
																									ignoreCase: false,
																								},
																								&charClassMatcher{
																									pos:             position{line: 230, col: 22, offset: 6463},
																									val:             "[\\r\\n]",
																									chars:           []rune{'\r', '\n'},
																									basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, false, true, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
																									ignoreCase:      false,
																									inverted:        false,
																								},
																							},
																						},
																					},
																				},
																			},
																			&labeledExpr{
																				pos:   position{line: 185, col: 23, offset: 5436},
																				label: "doctest",
																				expr: &actionExpr{
																					pos: position{line: 191, col: 3, offset: 5529},
																					run: (*parser).callonGrammar105,
																					expr: &seqExpr{
																						pos: position{line: 191, col: 3, offset: 5529},
																						exprs: []interface{}{
																							&labeledExpr{
																								pos:   position{line: 191, col: 3, offset: 5529},
																								label: "first",
																								expr: &actionExpr{
																									pos: position{line: 208, col: 21, offset: 5980},
																									run: (*parser).callonGrammar108,
																									expr: &seqExpr{
																										pos: position{line: 208, col: 21, offset: 5980},
																										exprs: []interface{}{
																											&labeledExpr{
																												pos:   position{line: 225, col: 11, offset: 6364},
																												label: "indent",
																												expr: &zeroOrMoreExpr{
																													pos: position{line: 225, col: 18, offset: 6371},
																													expr: &charClassMatcher{
																														pos:             position{line: 232, col: 16, offset: 6493},
																														val:             "[ \\t]",
																														chars:           []rune{' ', '\t'},
																														basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
																														ignoreCase:      false,
																														inverted:        false,
																													},
																												},
																											},
																											&stateCodeExpr{
																												pos: position{line: 226, col: 3, offset: 6386},
																												run: (*parser).callonGrammar113,
																											},
																											&labeledExpr{
																												pos:   position{line: 208, col: 28, offset: 5987},
																												label: "text",
																												expr: &seqExpr{
																													pos: position{line: 208, col: 35, offset: 5994},
																													exprs: []interface{}{
																														&litMatcher{
																															pos:        position{line: 208, col: 35, offset: 5994},
																															val:        ">>>",
																															ignoreCase: false,
																														},
																														&charClassMatcher{
																															pos:             position{line: 233, col: 15, offset: 6513},
																															val:             "[ \\t\\f]",
																															chars:           []rune{' ', '\t', '\f'},
																															basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, true, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
																															ignoreCase:      false,
																															inverted:        false,
																														},
																														&zeroOrMoreExpr{
																															pos: position{line: 208, col: 52, offset: 6011},
																															expr: &seqExpr{
																																pos: position{line: 208, col: 54, offset: 6013},
																																exprs: []interface{}{
																																	&notExpr{
																																		pos: position{line: 208, col: 54, offset: 6013},
																																		expr: &choiceExpr{
																																			pos: position{line: 230, col: 10, offset: 6451},
																																			alternatives: []interface{}{
																																				&litMatcher{
																																					pos:        position{line: 230, col: 10, offset: 6451},
																																					val:        "\r\n\n\n\n\n\n\n\n\n\n",
																																					ignoreCase: false,
																																				},
																																				&charClassMatcher{
																																					pos:             position{line: 230, col: 22, offset: 6463},
																																					val:             "[\\r\\n]",
																																					chars:           []rune{'\r', '\n'},
																																					basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, false, true, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
																																					ignoreCase:      false,
																																					inverted:        false,
																																				},
																																			},
																																		},
																																	},
																																	&anyMatcher{
																																		line: 208, col: 59, offset: 6018,
																																	},
																																},
																															},
																														},
																													},
																												},
																											},
																											&choiceExpr{
																												pos: position{line: 230, col: 10, offset: 6451},
																												alternatives: []interface{}{
																													&litMatcher{
																														pos:        position{line: 230, col: 10, offset: 6451},
																														val:        "\r\n\n\n\n\n\n\n\n\n\n",
																														ignoreCase: false,
																													},
																													&charClassMatcher{
																														pos:             position{line: 230, col: 22, offset: 6463},
																														val:             "[\\r\\n]",
																														chars:           []rune{'\r', '\n'},
																														basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, false, true, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
																														ignoreCase:      false,
																														inverted:        false,
																													},
																												},
																											},
																										},
																									},
																								},
																							},
																							&stateCodeExpr{
																								pos: position{line: 192, col: 3, offset: 5554},
																								run: (*parser).callonGrammar128,
																							},
																							&labeledExpr{
																								pos:   position{line: 196, col: 3, offset: 5665},
																								label: "rest",
																								expr: &zeroOrMoreExpr{
																									pos: position{line: 196, col: 8, offset: 5670},
																									expr: &seqExpr{
																										pos: position{line: 197, col: 5, offset: 5676},
																										exprs: []interface{}{
																											&labeledExpr{
																												pos:   position{line: 197, col: 5, offset: 5676},
																												label: "line",
																												expr: &actionExpr{
																													pos: position{line: 213, col: 17, offset: 6111},
																													run: (*parser).callonGrammar133,
																													expr: &seqExpr{
																														pos: position{line: 213, col: 17, offset: 6111},
																														exprs: []interface{}{
																															&labeledExpr{
																																pos:   position{line: 225, col: 11, offset: 6364},
																																label: "indent",
																																expr: &zeroOrMoreExpr{
																																	pos: position{line: 225, col: 18, offset: 6371},
																																	expr: &charClassMatcher{
																																		pos:             position{line: 232, col: 16, offset: 6493},
																																		val:             "[ \\t]",
																																		chars:           []rune{' ', '\t'},
																																		basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
																																		ignoreCase:      false,
																																		inverted:        false,
																																	},
																																},
																															},
																															&stateCodeExpr{
																																pos: position{line: 226, col: 3, offset: 6386},
																																run: (*parser).callonGrammar138,
																															},
																															&labeledExpr{
																																pos:   position{line: 213, col: 24, offset: 6118},
																																label: "text",
																																expr: &oneOrMoreExpr{
																																	pos: position{line: 213, col: 29, offset: 6123},
																																	expr: &seqExpr{
																																		pos: position{line: 213, col: 31, offset: 6125},
																																		exprs: []interface{}{
																																			&notExpr{
																																				pos: position{line: 213, col: 31, offset: 6125},
																																				expr: &choiceExpr{
																																					pos: position{line: 230, col: 10, offset: 6451},
																																					alternatives: []interface{}{
																																						&litMatcher{
																																							pos:        position{line: 230, col: 10, offset: 6451},
																																							val:        "\r\n\n\n\n\n\n\n\n\n\n",
																																							ignoreCase: false,
																																						},
																																						&charClassMatcher{
																																							pos:             position{line: 230, col: 22, offset: 6463},
																																							val:             "[\\r\\n]",
																																							chars:           []rune{'\r', '\n'},
																																							basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, false, true, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
																																							ignoreCase:      false,
																																							inverted:        false,
																																						},
																																					},
																																				},
																																			},
																																			&anyMatcher{
																																				line: 213, col: 36, offset: 6130,
																																			},
																																		},
																																	},
																																},
																															},
																															&choiceExpr{
																																pos: position{line: 230, col: 10, offset: 6451},
																																alternatives: []interface{}{
																																	&litMatcher{
																																		pos:        position{line: 230, col: 10, offset: 6451},
																																		val:        "\r\n\n\n\n\n\n\n\n\n\n",
																																		ignoreCase: false,
																																	},
																																	&charClassMatcher{
																																		pos:             position{line: 230, col: 22, offset: 6463},
																																		val:             "[\\r\\n]",
																																		chars:           []rune{'\r', '\n'},
																																		basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, false, true, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
																																		ignoreCase:      false,
																																		inverted:        false,
																																	},
																																},
																															},
																														},
																													},
																												},
																											},
																											&andCodeExpr{
																												pos: position{line: 198, col: 5, offset: 5698},
																												run: (*parser).callonGrammar150,
																											},
																										},
																									},
																								},
																							},
																						},
																					},
																				},
																			},
																			&choiceExpr{
																				pos: position{line: 185, col: 46, offset: 5459},
																				alternatives: []interface{}{
																					&andExpr{
																						pos: position{line: 185, col: 46, offset: 5459},
																						expr: &seqExpr{
																							pos: position{line: 234, col: 14, offset: 6534},
																							exprs: []interface{}{
																								&zeroOrMoreExpr{
																									pos: position{line: 234, col: 14, offset: 6534},
																									expr: &charClassMatcher{
																										pos:             position{line: 233, col: 15, offset: 6513},
																										val:             "[ \\t\\f]",
																										chars:           []rune{' ', '\t', '\f'},
																										basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, true, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
																										ignoreCase:      false,
																										inverted:        false,
																									},
																								},
																								&choiceExpr{
																									pos: position{line: 230, col: 10, offset: 6451},
																									alternatives: []interface{}{
																										&litMatcher{
																											pos:        position{line: 230, col: 10, offset: 6451},
																											val:        "\r\n\n\n\n\n\n\n\n\n\n",
																											ignoreCase: false,
																										},
																										&charClassMatcher{
																											pos:             position{line: 230, col: 22, offset: 6463},
																											val:             "[\\r\\n]",
																											chars:           []rune{'\r', '\n'},
																											basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, false, true, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
																											ignoreCase:      false,
																											inverted:        false,
																										},
																									},
																								},
																							},
																						},
																					},
																					&notExpr{
																						pos: position{line: 237, col: 8, offset: 6574},
																						expr: &anyMatcher{
																							line: 237, col: 9, offset: 6575,
																						},
																					},
																				},
																			},
																		},
																	},
																},
															},
															&stateCodeExpr{
																pos: position{line: 39, col: 19, offset: 953},
																run: (*parser).callonGrammar161,
															},
														},
													},
												},
												&actionExpr{
													pos: position{line: 40, col: 7, offset: 1026},
													run: (*parser).callonGrammar162,
													expr: &seqExpr{
														pos: position{line: 40, col: 7, offset: 1026},
														exprs: []interface{}{
															&labeledExpr{
																pos:   position{line: 40, col: 7, offset: 1026},
																label: "l",
																expr: &actionExpr{
																	pos: position{line: 99, col: 9, offset: 2666},
																	run: (*parser).callonGrammar165,
																	expr: &seqExpr{
																		pos: position{line: 99, col: 9, offset: 2666},
																		exprs: []interface{}{
																			&zeroOrMoreExpr{
																				pos: position{line: 235, col: 6, offset: 6555},
																				expr: &seqExpr{
																					pos: position{line: 234, col: 14, offset: 6534},
																					exprs: []interface{}{
																						&zeroOrMoreExpr{
																							pos: position{line: 234, col: 14, offset: 6534},
																							expr: &charClassMatcher{
																								pos:             position{line: 233, col: 15, offset: 6513},
																								val:             "[ \\t\\f]",
																								chars:           []rune{' ', '\t', '\f'},
																								basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, true, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
																								ignoreCase:      false,
																								inverted:        false,
																							},
																						},
																						&choiceExpr{
																							pos: position{line: 230, col: 10, offset: 6451},
																							alternatives: []interface{}{
																								&litMatcher{
																									pos:        position{line: 230, col: 10, offset: 6451},
																									val:        "\r\n\n\n\n\n\n\n\n\n\n",
																									ignoreCase: false,
																								},
																								&charClassMatcher{
																									pos:             position{line: 230, col: 22, offset: 6463},
																									val:             "[\\r\\n]",
																									chars:           []rune{'\r', '\n'},
																									basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, false, true, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
																									ignoreCase:      false,
																									inverted:        false,
																								},
																							},
																						},
																					},
																				},
																			},
																			&labeledExpr{
																				pos:   position{line: 225, col: 11, offset: 6364},
																				label: "indent",
																				expr: &zeroOrMoreExpr{
																					pos: position{line: 225, col: 18, offset: 6371},
																					expr: &charClassMatcher{
																						pos:             position{line: 232, col: 16, offset: 6493},
																						val:             "[ \\t]",
																						chars:           []rune{' ', '\t'},
																						basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
																						ignoreCase:      false,
																						inverted:        false,
																					},
																				},
																			},
																			&stateCodeExpr{
																				pos: position{line: 226, col: 3, offset: 6386},
																				run: (*parser).callonGrammar177,
																			},
																			&labeledExpr{
																				pos:   position{line: 99, col: 18, offset: 2675},
																				label: "bullet",
																				expr: &actionExpr{
																					pos: position{line: 108, col: 15, offset: 2948},
																					run: (*parser).callonGrammar179,
																					expr: &labeledExpr{
																						pos:   position{line: 108, col: 15, offset: 2948},
																						label: "bullet",
																						expr: &choiceExpr{
																							pos: position{line: 108, col: 24, offset: 2957},
																							alternatives: []interface{}{
																								&actionExpr{
																									pos: position{line: 113, col: 22, offset: 3052},
																									run: (*parser).callonGrammar182,
																									expr: &oneOrMoreExpr{
																										pos: position{line: 113, col: 22, offset: 3052},
																										expr: &seqExpr{
																											pos: position{line: 113, col: 24, offset: 3054},
																											exprs: []interface{}{
																												&oneOrMoreExpr{
																													pos: position{line: 219, col: 11, offset: 6226},
																													expr: &charClassMatcher{
																														pos:             position{line: 218, col: 10, offset: 6210},
																														val:             "[0-9]",
																														ranges:          []rune{'0', '9'},
																														basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, true, true, true, true, true, true, true, true, true, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
																														ignoreCase:      false,
																														inverted:        false,
																													},
																												},
																												&litMatcher{
																													pos:        position{line: 220, col: 8, offset: 6240},
																													val:        ".",
																													ignoreCase: false,
																												},
																											},
																										},
																									},
																								},
																								&actionExpr{
																									pos: position{line: 118, col: 24, offset: 3131},
																									run: (*parser).callonGrammar188,
																									expr: &litMatcher{
																										pos:        position{line: 118, col: 24, offset: 3131},
																										val:        "-",
																										ignoreCase: false,
																									},
																								},
																							},
																						},
																					},
																				},
																			},
																			&labeledExpr{
																				pos:   position{line: 99, col: 36, offset: 2693},
																				label: "text",
																				expr: &zeroOrOneExpr{
																					pos: position{line: 99, col: 41, offset: 2698},
																					expr: &seqExpr{
																						pos: position{line: 99, col: 43, offset: 2700},
																						exprs: []interface{}{
																							&charClassMatcher{
																								pos:             position{line: 233, col: 15, offset: 6513},
																								val:             "[ \\t\\f]",
																								chars:           []rune{' ', '\t', '\f'},
																								basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, true, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
																								ignoreCase:      false,
																								inverted:        false,
																							},
																							&zeroOrMoreExpr{
																								pos: position{line: 99, col: 54, offset: 2711},
																								expr: &seqExpr{
																									pos: position{line: 99, col: 56, offset: 2713},
																									exprs: []interface{}{
																										&notExpr{
																											pos: position{line: 99, col: 56, offset: 2713},
																											expr: &choiceExpr{
																												pos: position{line: 230, col: 10, offset: 6451},
																												alternatives: []interface{}{
																													&litMatcher{
																														pos:        position{line: 230, col: 10, offset: 6451},
																														val:        "\r\n\n\n\n\n\n\n\n\n\n",
																														ignoreCase: false,
																													},
																													&charClassMatcher{
																														pos:             position{line: 230, col: 22, offset: 6463},
																														val:             "[\\r\\n]",
																														chars:           []rune{'\r', '\n'},
																														basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, false, true, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
																														ignoreCase:      false,
																														inverted:        false,
																													},
																												},
																											},
																										},
																										&anyMatcher{
																											line: 99, col: 61, offset: 2718,
																										},
																									},
																								},
																							},
																						},
																					},
																				},
																			},
																			&choiceExpr{
																				pos: position{line: 230, col: 10, offset: 6451},
																				alternatives: []interface{}{
																					&litMatcher{
																						pos:        position{line: 230, col: 10, offset: 6451},
																						val:        "\r\n\n\n\n\n\n\n\n\n\n",
																						ignoreCase: false,
																					},
																					&charClassMatcher{
																						pos:             position{line: 230, col: 22, offset: 6463},
																						val:             "[\\r\\n]",
																						chars:           []rune{'\r', '\n'},
																						basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, false, true, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
																						ignoreCase:      false,
																						inverted:        false,
																					},
																				},
																			},
																			&labeledExpr{
																				pos:   position{line: 99, col: 73, offset: 2730},
																				label: "blank",
																				expr: &zeroOrOneExpr{
																					pos: position{line: 99, col: 79, offset: 2736},
																					expr: &actionExpr{
																						pos: position{line: 99, col: 81, offset: 2738},
																						run: (*parser).callonGrammar206,
																						expr: &andExpr{
																							pos: position{line: 99, col: 81, offset: 2738},
																							expr: &seqExpr{
																								pos: position{line: 234, col: 14, offset: 6534},
																								exprs: []interface{}{
																									&zeroOrMoreExpr{
																										pos: position{line: 234, col: 14, offset: 6534},
																										expr: &charClassMatcher{
																											pos:             position{line: 233, col: 15, offset: 6513},
																											val:             "[ \\t\\f]",
																											chars:           []rune{' ', '\t', '\f'},
																											basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, true, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
																											ignoreCase:      false,
																											inverted:        false,
																										},
																									},
																									&choiceExpr{
																										pos: position{line: 230, col: 10, offset: 6451},
																										alternatives: []interface{}{
																											&litMatcher{
																												pos:        position{line: 230, col: 10, offset: 6451},
																												val:        "\r\n\n\n\n\n\n\n\n\n\n",
																												ignoreCase: false,
																											},
																											&charClassMatcher{
																												pos:             position{line: 230, col: 22, offset: 6463},
																												val:             "[\\r\\n]",
																												chars:           []rune{'\r', '\n'},
																												basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, false, true, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
																												ignoreCase:      false,
																												inverted:        false,
																											},
																										},
																									},
																								},
																							},
																						},
																					},
																				},
																			},
																		},
																	},
																},
															},
															&stateCodeExpr{
																pos: position{line: 40, col: 19, offset: 1038},
																run: (*parser).callonGrammar214,
															},
														},
													},
												},
												&actionExpr{
													pos: position{line: 41, col: 7, offset: 1101},
													run: (*parser).callonGrammar215,
													expr: &seqExpr{
														pos: position{line: 41, col: 7, offset: 1101},
														exprs: []interface{}{
															&labeledExpr{
																pos:   position{line: 41, col: 7, offset: 1101},
																label: "f",
																expr: &actionExpr{
																	pos: position{line: 131, col: 10, offset: 3545},
																	run: (*parser).callonGrammar218,
																	expr: &seqExpr{
																		pos: position{line: 131, col: 10, offset: 3545},
																		exprs: []interface{}{
																			&zeroOrMoreExpr{
																				pos: position{line: 235, col: 6, offset: 6555},
																				expr: &seqExpr{
																					pos: position{line: 234, col: 14, offset: 6534},
																					exprs: []interface{}{
																						&zeroOrMoreExpr{
																							pos: position{line: 234, col: 14, offset: 6534},
																							expr: &charClassMatcher{
																								pos:             position{line: 233, col: 15, offset: 6513},
																								val:             "[ \\t\\f]",
																								chars:           []rune{' ', '\t', '\f'},
																								basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, true, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
																								ignoreCase:      false,
																								inverted:        false,
																							},
																						},
																						&choiceExpr{
																							pos: position{line: 230, col: 10, offset: 6451},
																							alternatives: []interface{}{
																								&litMatcher{
																									pos:        position{line: 230, col: 10, offset: 6451},
																									val:        "\r\n\n\n\n\n\n\n\n\n\n",
																									ignoreCase: false,
																								},
																								&charClassMatcher{
																									pos:             position{line: 230, col: 22, offset: 6463},
																									val:             "[\\r\\n]",
																									chars:           []rune{'\r', '\n'},
																									basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, false, true, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
																									ignoreCase:      false,
																									inverted:        false,
																								},
																							},
																						},
																					},
																				},
																			},
																			&labeledExpr{
																				pos:   position{line: 225, col: 11, offset: 6364},
																				label: "indent",
																				expr: &zeroOrMoreExpr{
																					pos: position{line: 225, col: 18, offset: 6371},
																					expr: &charClassMatcher{
																						pos:             position{line: 232, col: 16, offset: 6493},
																						val:             "[ \\t]",
																						chars:           []rune{' ', '\t'},
																						basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
																						ignoreCase:      false,
																						inverted:        false,
																					},
																				},
																			},
																			&stateCodeExpr{
																				pos: position{line: 226, col: 3, offset: 6386},
																				run: (*parser).callonGrammar230,
																			},
																			&labeledExpr{
																				pos:   position{line: 131, col: 19, offset: 3554},
																				label: "tag",
																				expr: &actionExpr{
																					pos: position{line: 148, col: 13, offset: 4383},
																					run: (*parser).callonGrammar232,
																					expr: &seqExpr{
																						pos: position{line: 148, col: 13, offset: 4383},
																						exprs: []interface{}{
																							&litMatcher{
																								pos:        position{line: 148, col: 13, offset: 4383},
																								val:        "@",
																								ignoreCase: false,
																							},
																							&labeledExpr{
																								pos:   position{line: 148, col: 17, offset: 4387},
																								label: "field",
																								expr: &actionExpr{
																									pos: position{line: 153, col: 15, offset: 4539},
																									run: (*parser).callonGrammar236,
																									expr: &oneOrMoreExpr{
																										pos: position{line: 153, col: 15, offset: 4539},
																										expr: &charClassMatcher{
																											pos:             position{line: 153, col: 15, offset: 4539},
																											val:             "[^:\\pWhite_Space]",
																											chars:           []rune{':'},
																											classes:         []*unicode.RangeTable{rangeTable("White_Space")},
																											basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, true, true, true, true, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
																											ignoreCase:      false,
																											inverted:        true,
																										},
																									},
																								},
																							},
																							&labeledExpr{
																								pos:   position{line: 148, col: 34, offset: 4404},
																								label: "arg",
																								expr: &zeroOrOneExpr{
																									pos: position{line: 148, col: 38, offset: 4408},
																									expr: &seqExpr{
																										pos: position{line: 148, col: 40, offset: 4410},
																										exprs: []interface{}{
																											&oneOrMoreExpr{
																												pos: position{line: 148, col: 40, offset: 4410},
																												expr: &charClassMatcher{
																													pos:             position{line: 233, col: 15, offset: 6513},
																													val:             "[ \\t\\f]",
																													chars:           []rune{' ', '\t', '\f'},
																													basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, true, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
																													ignoreCase:      false,
																													inverted:        false,
																												},
																											},
																											&actionExpr{
																												pos: position{line: 153, col: 15, offset: 4539},
																												run: (*parser).callonGrammar244,
																												expr: &oneOrMoreExpr{
																													pos: position{line: 153, col: 15, offset: 4539},
																													expr: &charClassMatcher{
																														pos:             position{line: 153, col: 15, offset: 4539},
																														val:             "[^:\\pWhite_Space]",
																														chars:           []rune{':'},
																														classes:         []*unicode.RangeTable{rangeTable("White_Space")},
																														basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, true, true, true, true, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
																														ignoreCase:      false,
																														inverted:        true,
																													},
																												},
																											},
																										},
																									},
																								},
																							},
																							&zeroOrMoreExpr{
																								pos: position{line: 148, col: 66, offset: 4436},
																								expr: &charClassMatcher{
																									pos:             position{line: 233, col: 15, offset: 6513},
																									val:             "[ \\t\\f]",
																									chars:           []rune{' ', '\t', '\f'},
																									basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, true, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
																									ignoreCase:      false,
																									inverted:        false,
																								},
																							},
																							&litMatcher{
																								pos:        position{line: 148, col: 78, offset: 4448},
																								val:        ":",
																								ignoreCase: false,
																							},
																						},
																					},
																				},
																			},
																			&labeledExpr{
																				pos:   position{line: 131, col: 32, offset: 3567},
																				label: "text",
																				expr: &zeroOrMoreExpr{
																					pos: position{line: 131, col: 37, offset: 3572},
																					expr: &seqExpr{
																						pos: position{line: 131, col: 39, offset: 3574},
																						exprs: []interface{}{
																							&notExpr{
																								pos: position{line: 131, col: 39, offset: 3574},
																								expr: &choiceExpr{
																									pos: position{line: 230, col: 10, offset: 6451},
																									alternatives: []interface{}{
																										&litMatcher{
																											pos:        position{line: 230, col: 10, offset: 6451},
																											val:        "\r\n\n\n\n\n\n\n\n\n\n",
																											ignoreCase: false,
																										},
																										&charClassMatcher{
																											pos:             position{line: 230, col: 22, offset: 6463},
																											val:             "[\\r\\n]",
																											chars:           []rune{'\r', '\n'},
																											basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, false, true, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
																											ignoreCase:      false,
																											inverted:        false,
																										},
																									},
																								},
																							},
																							&anyMatcher{
																								line: 131, col: 44, offset: 3579,
																							},
																						},
																					},
																				},
																			},
																			&choiceExpr{
																				pos: position{line: 230, col: 10, offset: 6451},
																				alternatives: []interface{}{
																					&litMatcher{
																						pos:        position{line: 230, col: 10, offset: 6451},
																						val:        "\r\n\n\n\n\n\n\n\n\n\n",
																						ignoreCase: false,
																					},
																					&charClassMatcher{
																						pos:             position{line: 230, col: 22, offset: 6463},
																						val:             "[\\r\\n]",
																						chars:           []rune{'\r', '\n'},
																						basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, false, true, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
																						ignoreCase:      false,
																						inverted:        false,
																					},
																				},
																			},
																			&labeledExpr{
																				pos:   position{line: 131, col: 53, offset: 3588},
																				label: "blank",
																				expr: &zeroOrOneExpr{
																					pos: position{line: 131, col: 59, offset: 3594},
																					expr: &actionExpr{
																						pos: position{line: 131, col: 61, offset: 3596},
																						run: (*parser).callonGrammar263,
																						expr: &andExpr{
																							pos: position{line: 131, col: 61, offset: 3596},
																							expr: &seqExpr{
																								pos: position{line: 234, col: 14, offset: 6534},
																								exprs: []interface{}{
																									&zeroOrMoreExpr{
																										pos: position{line: 234, col: 14, offset: 6534},
																										expr: &charClassMatcher{
																											pos:             position{line: 233, col: 15, offset: 6513},
																											val:             "[ \\t\\f]",
																											chars:           []rune{' ', '\t', '\f'},
																											basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, true, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
																											ignoreCase:      false,
																											inverted:        false,
																										},
																									},
																									&choiceExpr{
																										pos: position{line: 230, col: 10, offset: 6451},
																										alternatives: []interface{}{
																											&litMatcher{
																												pos:        position{line: 230, col: 10, offset: 6451},
																												val:        "\r\n\n\n\n\n\n\n\n\n\n",
																												ignoreCase: false,
																											},
																											&charClassMatcher{
																												pos:             position{line: 230, col: 22, offset: 6463},
																												val:             "[\\r\\n]",
																												chars:           []rune{'\r', '\n'},
																												basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, false, true, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
																												ignoreCase:      false,
																												inverted:        false,
																											},
																										},
																									},
																								},
																							},
																						},
																					},
																				},
																			},
																		},
																	},
																},
															},
															&stateCodeExpr{
																pos: position{line: 41, col: 19, offset: 1113},
																run: (*parser).callonGrammar271,
															},
														},
													},
												},
												&actionExpr{
													pos: position{line: 42, col: 7, offset: 1178},
													run: (*parser).callonGrammar272,
													expr: &seqExpr{
														pos: position{line: 42, col: 7, offset: 1178},
														exprs: []interface{}{
															&labeledExpr{
																pos:   position{line: 42, col: 7, offset: 1178},
																label: "p",
																expr: &actionExpr{
																	pos: position{line: 163, col: 3, offset: 4806},
																	run: (*parser).callonGrammar275,
																	expr: &seqExpr{
																		pos: position{line: 163, col: 3, offset: 4806},
																		exprs: []interface{}{
																			&zeroOrMoreExpr{
																				pos: position{line: 235, col: 6, offset: 6555},
																				expr: &seqExpr{
																					pos: position{line: 234, col: 14, offset: 6534},
																					exprs: []interface{}{
																						&zeroOrMoreExpr{
																							pos: position{line: 234, col: 14, offset: 6534},
																							expr: &charClassMatcher{
																								pos:             position{line: 233, col: 15, offset: 6513},
																								val:             "[ \\t\\f]",
																								chars:           []rune{' ', '\t', '\f'},
																								basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, true, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
																								ignoreCase:      false,
																								inverted:        false,
																							},
																						},
																						&choiceExpr{
																							pos: position{line: 230, col: 10, offset: 6451},
																							alternatives: []interface{}{
																								&litMatcher{
																									pos:        position{line: 230, col: 10, offset: 6451},
																									val:        "\r\n\n\n\n\n\n\n\n\n\n",
																									ignoreCase: false,
																								},
																								&charClassMatcher{
																									pos:             position{line: 230, col: 22, offset: 6463},
																									val:             "[\\r\\n]",
																									chars:           []rune{'\r', '\n'},
																									basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, false, true, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
																									ignoreCase:      false,
																									inverted:        false,
																								},
																							},
																						},
																					},
																				},
																			},
																			&labeledExpr{
																				pos:   position{line: 163, col: 5, offset: 4808},
																				label: "first",
																				expr: &actionExpr{
																					pos: position{line: 213, col: 17, offset: 6111},
																					run: (*parser).callonGrammar285,
																					expr: &seqExpr{
																						pos: position{line: 213, col: 17, offset: 6111},
																						exprs: []interface{}{
																							&labeledExpr{
																								pos:   position{line: 225, col: 11, offset: 6364},
																								label: "indent",
																								expr: &zeroOrMoreExpr{
																									pos: position{line: 225, col: 18, offset: 6371},
																									expr: &charClassMatcher{
																										pos:             position{line: 232, col: 16, offset: 6493},
																										val:             "[ \\t]",
																										chars:           []rune{' ', '\t'},
																										basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
																										ignoreCase:      false,
																										inverted:        false,
																									},
																								},
																							},
																							&stateCodeExpr{
																								pos: position{line: 226, col: 3, offset: 6386},
																								run: (*parser).callonGrammar290,
																							},
																							&labeledExpr{
																								pos:   position{line: 213, col: 24, offset: 6118},
																								label: "text",
																								expr: &oneOrMoreExpr{
																									pos: position{line: 213, col: 29, offset: 6123},
																									expr: &seqExpr{
																										pos: position{line: 213, col: 31, offset: 6125},
																										exprs: []interface{}{
																											&notExpr{
																												pos: position{line: 213, col: 31, offset: 6125},
																												expr: &choiceExpr{
																													pos: position{line: 230, col: 10, offset: 6451},
																													alternatives: []interface{}{
																														&litMatcher{
																															pos:        position{line: 230, col: 10, offset: 6451},
																															val:        "\r\n\n\n\n\n\n\n\n\n\n",
																															ignoreCase: false,
																														},
																														&charClassMatcher{
																															pos:             position{line: 230, col: 22, offset: 6463},
																															val:             "[\\r\\n]",
																															chars:           []rune{'\r', '\n'},
																															basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, false, true, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
																															ignoreCase:      false,
																															inverted:        false,
																														},
																													},
																												},
																											},
																											&anyMatcher{
																												line: 213, col: 36, offset: 6130,
																											},
																										},
																									},
																								},
																							},
																							&choiceExpr{
																								pos: position{line: 230, col: 10, offset: 6451},
																								alternatives: []interface{}{
																									&litMatcher{
																										pos:        position{line: 230, col: 10, offset: 6451},
																										val:        "\r\n\n\n\n\n\n\n\n\n\n",
																										ignoreCase: false,
																									},
																									&charClassMatcher{
																										pos:             position{line: 230, col: 22, offset: 6463},
																										val:             "[\\r\\n]",
																										chars:           []rune{'\r', '\n'},
																										basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, false, true, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
																										ignoreCase:      false,
																										inverted:        false,
																									},
																								},
																							},
																						},
																					},
																				},
																			},
																			&stateCodeExpr{
																				pos: position{line: 164, col: 3, offset: 4829},
																				run: (*parser).callonGrammar302,
																			},
																			&labeledExpr{
																				pos:   position{line: 168, col: 3, offset: 4944},
																				label: "rest",
																				expr: &zeroOrMoreExpr{
																					pos: position{line: 168, col: 8, offset: 4949},
																					expr: &seqExpr{
																						pos: position{line: 169, col: 5, offset: 4955},
																						exprs: []interface{}{
																							&labeledExpr{
																								pos:   position{line: 169, col: 5, offset: 4955},
																								label: "line",
																								expr: &actionExpr{
																									pos: position{line: 213, col: 17, offset: 6111},
																									run: (*parser).callonGrammar307,
																									expr: &seqExpr{
																										pos: position{line: 213, col: 17, offset: 6111},
																										exprs: []interface{}{
																											&labeledExpr{
																												pos:   position{line: 225, col: 11, offset: 6364},
																												label: "indent",
																												expr: &zeroOrMoreExpr{
																													pos: position{line: 225, col: 18, offset: 6371},
																													expr: &charClassMatcher{
																														pos:             position{line: 232, col: 16, offset: 6493},
																														val:             "[ \\t]",
																														chars:           []rune{' ', '\t'},
																														basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
																														ignoreCase:      false,
																														inverted:        false,
																													},
																												},
																											},
																											&stateCodeExpr{
																												pos: position{line: 226, col: 3, offset: 6386},
																												run: (*parser).callonGrammar312,
																											},
																											&labeledExpr{
																												pos:   position{line: 213, col: 24, offset: 6118},
																												label: "text",
																												expr: &oneOrMoreExpr{
																													pos: position{line: 213, col: 29, offset: 6123},
																													expr: &seqExpr{
																														pos: position{line: 213, col: 31, offset: 6125},
																														exprs: []interface{}{
																															&notExpr{
																																pos: position{line: 213, col: 31, offset: 6125},
																																expr: &choiceExpr{
																																	pos: position{line: 230, col: 10, offset: 6451},
																																	alternatives: []interface{}{
																																		&litMatcher{
																																			pos:        position{line: 230, col: 10, offset: 6451},
																																			val:        "\r\n\n\n\n\n\n\n\n\n\n",
																																			ignoreCase: false,
																																		},
																																		&charClassMatcher{
																																			pos:             position{line: 230, col: 22, offset: 6463},
																																			val:             "[\\r\\n]",
																																			chars:           []rune{'\r', '\n'},
																																			basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, false, true, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
																																			ignoreCase:      false,
																																			inverted:        false,
																																		},
																																	},
																																},
																															},
																															&anyMatcher{
																																line: 213, col: 36, offset: 6130,
																															},
																														},
																													},
																												},
																											},
																											&choiceExpr{
																												pos: position{line: 230, col: 10, offset: 6451},
																												alternatives: []interface{}{
																													&litMatcher{
																														pos:        position{line: 230, col: 10, offset: 6451},
																														val:        "\r\n\n\n\n\n\n\n\n\n\n",
																														ignoreCase: false,
																													},
																													&charClassMatcher{
																														pos:             position{line: 230, col: 22, offset: 6463},
																														val:             "[\\r\\n]",
																														chars:           []rune{'\r', '\n'},
																														basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, false, true, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
																														ignoreCase:      false,
																														inverted:        false,
																													},
																												},
																											},
																										},
																									},
																								},
																							},
																							&andCodeExpr{
																								pos: position{line: 170, col: 5, offset: 4977},
																								run: (*parser).callonGrammar324,
																							},
																						},
																					},
																				},
																			},
																		},
																	},
																},
															},
															&stateCodeExpr{
																pos: position{line: 42, col: 19, offset: 1190},
																run: (*parser).callonGrammar325,
															},
														},
													},
												},
											},
										},
									},
								},
							},
						},
						&zeroOrMoreExpr{
							pos: position{line: 235, col: 6, offset: 6555},
							expr: &seqExpr{
								pos: position{line: 234, col: 14, offset: 6534},
								exprs: []interface{}{
									&zeroOrMoreExpr{
										pos: position{line: 234, col: 14, offset: 6534},
										expr: &charClassMatcher{
											pos:             position{line: 233, col: 15, offset: 6513},
											val:             "[ \\t\\f]",
											chars:           []rune{' ', '\t', '\f'},
											basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, true, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
											ignoreCase:      false,
											inverted:        false,
										},
									},
									&choiceExpr{
										pos: position{line: 230, col: 10, offset: 6451},
										alternatives: []interface{}{
											&litMatcher{
												pos:        position{line: 230, col: 10, offset: 6451},
												val:        "\r\n\n\n\n\n\n\n\n\n\n",
												ignoreCase: false,
											},
											&charClassMatcher{
												pos:             position{line: 230, col: 22, offset: 6463},
												val:             "[\\r\\n]",
												chars:           []rune{'\r', '\n'},
												basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, false, true, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
												ignoreCase:      false,
												inverted:        false,
											},
										},
									},
								},
							},
						},
						&zeroOrMoreExpr{
							pos: position{line: 18, col: 19, offset: 384},
							expr: &charClassMatcher{
								pos:             position{line: 233, col: 15, offset: 6513},
								val:             "[ \\t\\f]",
								chars:           []rune{' ', '\t', '\f'},
								basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, true, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
								ignoreCase:      false,
								inverted:        false,
							},
						},
						&notExpr{
							pos: position{line: 237, col: 8, offset: 6574},
							expr: &anyMatcher{
								line: 237, col: 9, offset: 6575,
							},
						},
					},
				},
			},
		},
		{
			name: "TestInternalBlocks",
			pos:  position{line: 23, col: 1, offset: 459},
			expr: &actionExpr{
				pos: position{line: 24, col: 3, offset: 483},
				run: (*parser).callonTestInternalBlocks1,
				expr: &seqExpr{
					pos: position{line: 24, col: 3, offset: 483},
					exprs: []interface{}{
						&stateCodeExpr{
							pos: position{line: 24, col: 3, offset: 483},
							run: (*parser).callonTestInternalBlocks3,
						},
						&labeledExpr{
							pos:   position{line: 27, col: 3, offset: 516},
							label: "blocks",
							expr: &zeroOrMoreExpr{
								pos: position{line: 27, col: 10, offset: 523},
								expr: &actionExpr{
									pos: position{line: 36, col: 3, offset: 828},
									run: (*parser).callonTestInternalBlocks6,
									expr: &labeledExpr{
										pos:   position{line: 36, col: 3, offset: 828},
										label: "block",
										expr: &choiceExpr{
											pos: position{line: 37, col: 7, offset: 842},
											alternatives: []interface{}{
												&actionExpr{
													pos: position{line: 37, col: 7, offset: 842},
													run: (*parser).callonTestInternalBlocks9,
													expr: &seqExpr{
														pos: position{line: 37, col: 7, offset: 842},
														exprs: []interface{}{
															&labeledExpr{
																pos:   position{line: 37, col: 7, offset: 842},
																label: "lit",
																expr: &actionExpr{
																	pos: position{line: 55, col: 3, offset: 1550},
																	run: (*parser).callonTestInternalBlocks12,
																	expr: &seqExpr{
																		pos: position{line: 55, col: 3, offset: 1550},
																		exprs: []interface{}{
																			&andCodeExpr{
																				pos: position{line: 55, col: 3, offset: 1550},
																				run: (*parser).callonTestInternalBlocks14,
																			},
																			&labeledExpr{
																				pos:   position{line: 59, col: 3, offset: 1664},
																				label: "lines",
																				expr: &oneOrMoreExpr{
																					pos: position{line: 59, col: 9, offset: 1670},
																					expr: &seqExpr{
																						pos: position{line: 60, col: 5, offset: 1676},
																						exprs: []interface{}{
																							&zeroOrMoreExpr{
																								pos: position{line: 235, col: 6, offset: 6555},
																								expr: &seqExpr{
																									pos: position{line: 234, col: 14, offset: 6534},
																									exprs: []interface{}{
																										&zeroOrMoreExpr{
																											pos: position{line: 234, col: 14, offset: 6534},
																											expr: &charClassMatcher{
																												pos:             position{line: 233, col: 15, offset: 6513},
																												val:             "[ \\t\\f]",
																												chars:           []rune{' ', '\t', '\f'},
																												basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, true, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
																												ignoreCase:      false,
																												inverted:        false,
																											},
																										},
																										&choiceExpr{
																											pos: position{line: 230, col: 10, offset: 6451},
																											alternatives: []interface{}{
																												&litMatcher{
																													pos:        position{line: 230, col: 10, offset: 6451},
																													val:        "\r\n\n\n\n\n\n\n\n\n\n",
																													ignoreCase: false,
																												},
																												&charClassMatcher{
																													pos:             position{line: 230, col: 22, offset: 6463},
																													val:             "[\\r\\n]",
																													chars:           []rune{'\r', '\n'},
																													basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, false, true, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
																													ignoreCase:      false,
																													inverted:        false,
																												},
																											},
																										},
																									},
																								},
																							},
																							&labeledExpr{
																								pos:   position{line: 60, col: 7, offset: 1678},
																								label: "line",
																								expr: &actionExpr{
																									pos: position{line: 213, col: 17, offset: 6111},
																									run: (*parser).callonTestInternalBlocks26,
																									expr: &seqExpr{
																										pos: position{line: 213, col: 17, offset: 6111},
																										exprs: []interface{}{
																											&labeledExpr{
																												pos:   position{line: 225, col: 11, offset: 6364},
																												label: "indent",
																												expr: &zeroOrMoreExpr{
																													pos: position{line: 225, col: 18, offset: 6371},
																													expr: &charClassMatcher{
																														pos:             position{line: 232, col: 16, offset: 6493},
																														val:             "[ \\t]",
																														chars:           []rune{' ', '\t'},
																														basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
																														ignoreCase:      false,
																														inverted:        false,
																													},
																												},
																											},
																											&stateCodeExpr{
																												pos: position{line: 226, col: 3, offset: 6386},
																												run: (*parser).callonTestInternalBlocks31,
																											},
																											&labeledExpr{
																												pos:   position{line: 213, col: 24, offset: 6118},
																												label: "text",
																												expr: &oneOrMoreExpr{
																													pos: position{line: 213, col: 29, offset: 6123},
																													expr: &seqExpr{
																														pos: position{line: 213, col: 31, offset: 6125},
																														exprs: []interface{}{
																															&notExpr{
																																pos: position{line: 213, col: 31, offset: 6125},
																																expr: &choiceExpr{
																																	pos: position{line: 230, col: 10, offset: 6451},
																																	alternatives: []interface{}{
																																		&litMatcher{
																																			pos:        position{line: 230, col: 10, offset: 6451},
																																			val:        "\r\n\n\n\n\n\n\n\n\n\n",
																																			ignoreCase: false,
																																		},
																																		&charClassMatcher{
																																			pos:             position{line: 230, col: 22, offset: 6463},
																																			val:             "[\\r\\n]",
																																			chars:           []rune{'\r', '\n'},
																																			basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, false, true, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
																																			ignoreCase:      false,
																																			inverted:        false,
																																		},
																																	},
																																},
																															},
																															&anyMatcher{
																																line: 213, col: 36, offset: 6130,
																															},
																														},
																													},
																												},
																											},
																											&choiceExpr{
																												pos: position{line: 230, col: 10, offset: 6451},
																												alternatives: []interface{}{
																													&litMatcher{
																														pos:        position{line: 230, col: 10, offset: 6451},
																														val:        "\r\n\n\n\n\n\n\n\n\n\n",
																														ignoreCase: false,
																													},
																													&charClassMatcher{
																														pos:             position{line: 230, col: 22, offset: 6463},
																														val:             "[\\r\\n]",
																														chars:           []rune{'\r', '\n'},
																														basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, false, true, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
																														ignoreCase:      false,
																														inverted:        false,
																													},
																												},
																											},
																										},
																									},
																								},
																							},
																							&andCodeExpr{
																								pos: position{line: 61, col: 5, offset: 1700},
																								run: (*parser).callonTestInternalBlocks43,
																							},
																						},
																					},
																				},
																			},
																		},
																	},
																},
															},
															&stateCodeExpr{
																pos: position{line: 37, col: 19, offset: 854},
																run: (*parser).callonTestInternalBlocks44,
															},
														},
													},
												},
												&actionExpr{
													pos: position{line: 75, col: 3, offset: 2013},
													run: (*parser).callonTestInternalBlocks45,
													expr: &seqExpr{
														pos: position{line: 75, col: 3, offset: 2013},
														exprs: []interface{}{
															&zeroOrMoreExpr{
																pos: position{line: 235, col: 6, offset: 6555},
																expr: &seqExpr{
																	pos: position{line: 234, col: 14, offset: 6534},
																	exprs: []interface{}{
																		&zeroOrMoreExpr{
																			pos: position{line: 234, col: 14, offset: 6534},
																			expr: &charClassMatcher{
																				pos:             position{line: 233, col: 15, offset: 6513},
																				val:             "[ \\t\\f]",
																				chars:           []rune{' ', '\t', '\f'},
																				basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, true, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
																				ignoreCase:      false,
																				inverted:        false,
																			},
																		},
																		&choiceExpr{
																			pos: position{line: 230, col: 10, offset: 6451},
																			alternatives: []interface{}{
																				&litMatcher{
																					pos:        position{line: 230, col: 10, offset: 6451},
																					val:        "\r\n\n\n\n\n\n\n\n\n\n",
																					ignoreCase: false,
																				},
																				&charClassMatcher{
																					pos:             position{line: 230, col: 22, offset: 6463},
																					val:             "[\\r\\n]",
																					chars:           []rune{'\r', '\n'},
																					basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, false, true, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
																					ignoreCase:      false,
																					inverted:        false,
																				},
																			},
																		},
																	},
																},
															},
															&labeledExpr{
																pos:   position{line: 75, col: 5, offset: 2015},
																label: "header",
																expr: &actionExpr{
																	pos: position{line: 213, col: 17, offset: 6111},
																	run: (*parser).callonTestInternalBlocks55,
																	expr: &seqExpr{
																		pos: position{line: 213, col: 17, offset: 6111},
																		exprs: []interface{}{
																			&labeledExpr{
																				pos:   position{line: 225, col: 11, offset: 6364},
																				label: "indent",
																				expr: &zeroOrMoreExpr{
																					pos: position{line: 225, col: 18, offset: 6371},
																					expr: &charClassMatcher{
																						pos:             position{line: 232, col: 16, offset: 6493},
																						val:             "[ \\t]",
																						chars:           []rune{' ', '\t'},
																						basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
																						ignoreCase:      false,
																						inverted:        false,
																					},
																				},
																			},
																			&stateCodeExpr{
																				pos: position{line: 226, col: 3, offset: 6386},
																				run: (*parser).callonTestInternalBlocks60,
																			},
																			&labeledExpr{
																				pos:   position{line: 213, col: 24, offset: 6118},
																				label: "text",
																				expr: &oneOrMoreExpr{
																					pos: position{line: 213, col: 29, offset: 6123},
																					expr: &seqExpr{
																						pos: position{line: 213, col: 31, offset: 6125},
																						exprs: []interface{}{
																							&notExpr{
																								pos: position{line: 213, col: 31, offset: 6125},
																								expr: &choiceExpr{
																									pos: position{line: 230, col: 10, offset: 6451},
																									alternatives: []interface{}{
																										&litMatcher{
																											pos:        position{line: 230, col: 10, offset: 6451},
																											val:        "\r\n\n\n\n\n\n\n\n\n\n",
																											ignoreCase: false,
																										},
																										&charClassMatcher{
																											pos:             position{line: 230, col: 22, offset: 6463},
																											val:             "[\\r\\n]",
																											chars:           []rune{'\r', '\n'},
																											basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, false, true, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
																											ignoreCase:      false,
																											inverted:        false,
																										},
																									},
																								},
																							},
																							&anyMatcher{
																								line: 213, col: 36, offset: 6130,
																							},
																						},
																					},
																				},
																			},
																			&choiceExpr{
																				pos: position{line: 230, col: 10, offset: 6451},
																				alternatives: []interface{}{
																					&litMatcher{
																						pos:        position{line: 230, col: 10, offset: 6451},
																						val:        "\r\n\n\n\n\n\n\n\n\n\n",
																						ignoreCase: false,
																					},
																					&charClassMatcher{
																						pos:             position{line: 230, col: 22, offset: 6463},
																						val:             "[\\r\\n]",
																						chars:           []rune{'\r', '\n'},
																						basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, false, true, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
																						ignoreCase:      false,
																						inverted:        false,
																					},
																				},
																			},
																		},
																	},
																},
															},
															&labeledExpr{
																pos:   position{line: 75, col: 25, offset: 2035},
																label: "underline",
																expr: &actionExpr{
																	pos: position{line: 83, col: 18, offset: 2244},
																	run: (*parser).callonTestInternalBlocks73,
																	expr: &seqExpr{
																		pos: position{line: 83, col: 18, offset: 2244},
																		exprs: []interface{}{
																			&labeledExpr{
																				pos:   position{line: 225, col: 11, offset: 6364},
																				label: "indent",
																				expr: &zeroOrMoreExpr{
																					pos: position{line: 225, col: 18, offset: 6371},
																					expr: &charClassMatcher{
																						pos:             position{line: 232, col: 16, offset: 6493},
																						val:             "[ \\t]",
																						chars:           []rune{' ', '\t'},
																						basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
																						ignoreCase:      false,
																						inverted:        false,
																					},
																				},
																			},
																			&stateCodeExpr{
																				pos: position{line: 226, col: 3, offset: 6386},
																				run: (*parser).callonTestInternalBlocks78,
																			},
																			&labeledExpr{
																				pos:   position{line: 83, col: 25, offset: 2251},
																				label: "underline",
																				expr: &actionExpr{
																					pos: position{line: 88, col: 21, offset: 2337},
																					run: (*parser).callonTestInternalBlocks80,
																					expr: &choiceExpr{
																						pos: position{line: 88, col: 23, offset: 2339},
																						alternatives: []interface{}{
																							&oneOrMoreExpr{
																								pos: position{line: 88, col: 23, offset: 2339},
																								expr: &litMatcher{
																									pos:        position{line: 88, col: 23, offset: 2339},
																									val:        "=",
																									ignoreCase: false,
																								},
																							},
																							&oneOrMoreExpr{
																								pos: position{line: 88, col: 30, offset: 2346},
																								expr: &litMatcher{
																									pos:        position{line: 88, col: 30, offset: 2346},
																									val:        "-",
																									ignoreCase: false,
																								},
																							},
																							&oneOrMoreExpr{
																								pos: position{line: 88, col: 37, offset: 2353},
																								expr: &litMatcher{
																									pos:        position{line: 88, col: 37, offset: 2353},
																									val:        "~",
																									ignoreCase: false,
																								},
																							},
																						},
																					},
																				},
																			},
																			&choiceExpr{
																				pos: position{line: 230, col: 10, offset: 6451},
																				alternatives: []interface{}{
																					&litMatcher{
																						pos:        position{line: 230, col: 10, offset: 6451},
																						val:        "\r\n\n\n\n\n\n\n\n\n\n",
																						ignoreCase: false,
																					},
																					&charClassMatcher{
																						pos:             position{line: 230, col: 22, offset: 6463},
																						val:             "[\\r\\n]",
																						chars:           []rune{'\r', '\n'},
																						basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, false, true, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
																						ignoreCase:      false,
																						inverted:        false,
																					},
																				},
																			},
																		},
																	},
																},
															},
															&andCodeExpr{
																pos: position{line: 76, col: 3, offset: 2061},
																run: (*parser).callonTestInternalBlocks91,
															},
														},
													},
												},
												&actionExpr{
													pos: position{line: 39, col: 7, offset: 941},
													run: (*parser).callonTestInternalBlocks92,
													expr: &seqExpr{
														pos: position{line: 39, col: 7, offset: 941},
														exprs: []interface{}{
															&labeledExpr{
																pos:   position{line: 39, col: 7, offset: 941},
																label: "doc",
																expr: &actionExpr{
																	pos: position{line: 185, col: 12, offset: 5425},
																	run: (*parser).callonTestInternalBlocks95,
																	expr: &seqExpr{
																		pos: position{line: 185, col: 12, offset: 5425},
																		exprs: []interface{}{
																			&oneOrMoreExpr{
																				pos: position{line: 185, col: 12, offset: 5425},
																				expr: &seqExpr{
																					pos: position{line: 234, col: 14, offset: 6534},
																					exprs: []interface{}{
																						&zeroOrMoreExpr{
																							pos: position{line: 234, col: 14, offset: 6534},
																							expr: &charClassMatcher{
																								pos:             position{line: 233, col: 15, offset: 6513},
																								val:             "[ \\t\\f]",
																								chars:           []rune{' ', '\t', '\f'},
																								basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, true, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
																								ignoreCase:      false,
																								inverted:        false,
																							},
																						},
																						&choiceExpr{
																							pos: position{line: 230, col: 10, offset: 6451},
																							alternatives: []interface{}{
																								&litMatcher{
																									pos:        position{line: 230, col: 10, offset: 6451},
																									val:        "\r\n\n\n\n\n\n\n\n\n\n",
																									ignoreCase: false,
																								},
																								&charClassMatcher{
																									pos:             position{line: 230, col: 22, offset: 6463},
																									val:             "[\\r\\n]",
																									chars:           []rune{'\r', '\n'},
																									basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, false, true, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
																									ignoreCase:      false,
																									inverted:        false,
																								},
																							},
																						},
																					},
																				},
																			},
																			&labeledExpr{
																				pos:   position{line: 185, col: 23, offset: 5436},
																				label: "doctest",
																				expr: &actionExpr{
																					pos: position{line: 191, col: 3, offset: 5529},
																					run: (*parser).callonTestInternalBlocks105,
																					expr: &seqExpr{
																						pos: position{line: 191, col: 3, offset: 5529},
																						exprs: []interface{}{
																							&labeledExpr{
																								pos:   position{line: 191, col: 3, offset: 5529},
																								label: "first",
																								expr: &actionExpr{
																									pos: position{line: 208, col: 21, offset: 5980},
																									run: (*parser).callonTestInternalBlocks108,
																									expr: &seqExpr{
																										pos: position{line: 208, col: 21, offset: 5980},
																										exprs: []interface{}{
																											&labeledExpr{
																												pos:   position{line: 225, col: 11, offset: 6364},
																												label: "indent",
																												expr: &zeroOrMoreExpr{
																													pos: position{line: 225, col: 18, offset: 6371},
																													expr: &charClassMatcher{
																														pos:             position{line: 232, col: 16, offset: 6493},
																														val:             "[ \\t]",
																														chars:           []rune{' ', '\t'},
																														basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
																														ignoreCase:      false,
																														inverted:        false,
																													},
																												},
																											},
																											&stateCodeExpr{
																												pos: position{line: 226, col: 3, offset: 6386},
																												run: (*parser).callonTestInternalBlocks113,
																											},
																											&labeledExpr{
																												pos:   position{line: 208, col: 28, offset: 5987},
																												label: "text",
																												expr: &seqExpr{
																													pos: position{line: 208, col: 35, offset: 5994},
																													exprs: []interface{}{
																														&litMatcher{
																															pos:        position{line: 208, col: 35, offset: 5994},
																															val:        ">>>",
																															ignoreCase: false,
																														},
																														&charClassMatcher{
																															pos:             position{line: 233, col: 15, offset: 6513},
																															val:             "[ \\t\\f]",
																															chars:           []rune{' ', '\t', '\f'},
																															basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, true, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
																															ignoreCase:      false,
																															inverted:        false,
																														},
																														&zeroOrMoreExpr{
																															pos: position{line: 208, col: 52, offset: 6011},
																															expr: &seqExpr{
																																pos: position{line: 208, col: 54, offset: 6013},
																																exprs: []interface{}{
																																	&notExpr{
																																		pos: position{line: 208, col: 54, offset: 6013},
																																		expr: &choiceExpr{
																																			pos: position{line: 230, col: 10, offset: 6451},
																																			alternatives: []interface{}{
																																				&litMatcher{
																																					pos:        position{line: 230, col: 10, offset: 6451},
																																					val:        "\r\n\n\n\n\n\n\n\n\n\n",
																																					ignoreCase: false,
																																				},
																																				&charClassMatcher{
																																					pos:             position{line: 230, col: 22, offset: 6463},
																																					val:             "[\\r\\n]",
																																					chars:           []rune{'\r', '\n'},
																																					basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, false, true, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
																																					ignoreCase:      false,
																																					inverted:        false,
																																				},
																																			},
																																		},
																																	},
																																	&anyMatcher{
																																		line: 208, col: 59, offset: 6018,
																																	},
																																},
																															},
																														},
																													},
																												},
																											},
																											&choiceExpr{
																												pos: position{line: 230, col: 10, offset: 6451},
																												alternatives: []interface{}{
																													&litMatcher{
																														pos:        position{line: 230, col: 10, offset: 6451},
																														val:        "\r\n\n\n\n\n\n\n\n\n\n",
																														ignoreCase: false,
																													},
																													&charClassMatcher{
																														pos:             position{line: 230, col: 22, offset: 6463},
																														val:             "[\\r\\n]",
																														chars:           []rune{'\r', '\n'},
																														basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, false, true, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
																														ignoreCase:      false,
																														inverted:        false,
																													},
																												},
																											},
																										},
																									},
																								},
																							},
																							&stateCodeExpr{
																								pos: position{line: 192, col: 3, offset: 5554},
																								run: (*parser).callonTestInternalBlocks128,
																							},
																							&labeledExpr{
																								pos:   position{line: 196, col: 3, offset: 5665},
																								label: "rest",
																								expr: &zeroOrMoreExpr{
																									pos: position{line: 196, col: 8, offset: 5670},
																									expr: &seqExpr{
																										pos: position{line: 197, col: 5, offset: 5676},
																										exprs: []interface{}{
																											&labeledExpr{
																												pos:   position{line: 197, col: 5, offset: 5676},
																												label: "line",
																												expr: &actionExpr{
																													pos: position{line: 213, col: 17, offset: 6111},
																													run: (*parser).callonTestInternalBlocks133,
																													expr: &seqExpr{
																														pos: position{line: 213, col: 17, offset: 6111},
																														exprs: []interface{}{
																															&labeledExpr{
																																pos:   position{line: 225, col: 11, offset: 6364},
																																label: "indent",
																																expr: &zeroOrMoreExpr{
																																	pos: position{line: 225, col: 18, offset: 6371},
																																	expr: &charClassMatcher{
																																		pos:             position{line: 232, col: 16, offset: 6493},
																																		val:             "[ \\t]",
																																		chars:           []rune{' ', '\t'},
																																		basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
																																		ignoreCase:      false,
																																		inverted:        false,
																																	},
																																},
																															},
																															&stateCodeExpr{
																																pos: position{line: 226, col: 3, offset: 6386},
																																run: (*parser).callonTestInternalBlocks138,
																															},
																															&labeledExpr{
																																pos:   position{line: 213, col: 24, offset: 6118},
																																label: "text",
																																expr: &oneOrMoreExpr{
																																	pos: position{line: 213, col: 29, offset: 6123},
																																	expr: &seqExpr{
																																		pos: position{line: 213, col: 31, offset: 6125},
																																		exprs: []interface{}{
																																			&notExpr{
																																				pos: position{line: 213, col: 31, offset: 6125},
																																				expr: &choiceExpr{
																																					pos: position{line: 230, col: 10, offset: 6451},
																																					alternatives: []interface{}{
																																						&litMatcher{
																																							pos:        position{line: 230, col: 10, offset: 6451},
																																							val:        "\r\n\n\n\n\n\n\n\n\n\n",
																																							ignoreCase: false,
																																						},
																																						&charClassMatcher{
																																							pos:             position{line: 230, col: 22, offset: 6463},
																																							val:             "[\\r\\n]",
																																							chars:           []rune{'\r', '\n'},
																																							basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, false, true, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
																																							ignoreCase:      false,
																																							inverted:        false,
																																						},
																																					},
																																				},
																																			},
																																			&anyMatcher{
																																				line: 213, col: 36, offset: 6130,
																																			},
																																		},
																																	},
																																},
																															},
																															&choiceExpr{
																																pos: position{line: 230, col: 10, offset: 6451},
																																alternatives: []interface{}{
																																	&litMatcher{
																																		pos:        position{line: 230, col: 10, offset: 6451},
																																		val:        "\r\n\n\n\n\n\n\n\n\n\n",
																																		ignoreCase: false,
																																	},
																																	&charClassMatcher{
																																		pos:             position{line: 230, col: 22, offset: 6463},
																																		val:             "[\\r\\n]",
																																		chars:           []rune{'\r', '\n'},
																																		basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, false, true, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
																																		ignoreCase:      false,
																																		inverted:        false,
																																	},
																																},
																															},
																														},
																													},
																												},
																											},
																											&andCodeExpr{
																												pos: position{line: 198, col: 5, offset: 5698},
																												run: (*parser).callonTestInternalBlocks150,
																											},
																										},
																									},
																								},
																							},
																						},
																					},
																				},
																			},
																			&choiceExpr{
																				pos: position{line: 185, col: 46, offset: 5459},
																				alternatives: []interface{}{
																					&andExpr{
																						pos: position{line: 185, col: 46, offset: 5459},
																						expr: &seqExpr{
																							pos: position{line: 234, col: 14, offset: 6534},
																							exprs: []interface{}{
																								&zeroOrMoreExpr{
																									pos: position{line: 234, col: 14, offset: 6534},
																									expr: &charClassMatcher{
																										pos:             position{line: 233, col: 15, offset: 6513},
																										val:             "[ \\t\\f]",
																										chars:           []rune{' ', '\t', '\f'},
																										basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, true, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
																										ignoreCase:      false,
																										inverted:        false,
																									},
																								},
																								&choiceExpr{
																									pos: position{line: 230, col: 10, offset: 6451},
																									alternatives: []interface{}{
																										&litMatcher{
																											pos:        position{line: 230, col: 10, offset: 6451},
																											val:        "\r\n\n\n\n\n\n\n\n\n\n",
																											ignoreCase: false,
																										},
																										&charClassMatcher{
																											pos:             position{line: 230, col: 22, offset: 6463},
																											val:             "[\\r\\n]",
																											chars:           []rune{'\r', '\n'},
																											basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, false, true, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
																											ignoreCase:      false,
																											inverted:        false,
																										},
																									},
																								},
																							},
																						},
																					},
																					&notExpr{
																						pos: position{line: 237, col: 8, offset: 6574},
																						expr: &anyMatcher{
																							line: 237, col: 9, offset: 6575,
																						},
																					},
																				},
																			},
																		},
																	},
																},
															},
															&stateCodeExpr{
																pos: position{line: 39, col: 19, offset: 953},
																run: (*parser).callonTestInternalBlocks161,
															},
														},
													},
												},
												&actionExpr{
													pos: position{line: 40, col: 7, offset: 1026},
													run: (*parser).callonTestInternalBlocks162,
													expr: &seqExpr{
														pos: position{line: 40, col: 7, offset: 1026},
														exprs: []interface{}{
															&labeledExpr{
																pos:   position{line: 40, col: 7, offset: 1026},
																label: "l",
																expr: &actionExpr{
																	pos: position{line: 99, col: 9, offset: 2666},
																	run: (*parser).callonTestInternalBlocks165,
																	expr: &seqExpr{
																		pos: position{line: 99, col: 9, offset: 2666},
																		exprs: []interface{}{
																			&zeroOrMoreExpr{
																				pos: position{line: 235, col: 6, offset: 6555},
																				expr: &seqExpr{
																					pos: position{line: 234, col: 14, offset: 6534},
																					exprs: []interface{}{
																						&zeroOrMoreExpr{
																							pos: position{line: 234, col: 14, offset: 6534},
																							expr: &charClassMatcher{
																								pos:             position{line: 233, col: 15, offset: 6513},
																								val:             "[ \\t\\f]",
																								chars:           []rune{' ', '\t', '\f'},
																								basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, true, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
																								ignoreCase:      false,
																								inverted:        false,
																							},
																						},
																						&choiceExpr{
																							pos: position{line: 230, col: 10, offset: 6451},
																							alternatives: []interface{}{
																								&litMatcher{
																									pos:        position{line: 230, col: 10, offset: 6451},
																									val:        "\r\n\n\n\n\n\n\n\n\n\n",
																									ignoreCase: false,
																								},
																								&charClassMatcher{
																									pos:             position{line: 230, col: 22, offset: 6463},
																									val:             "[\\r\\n]",
																									chars:           []rune{'\r', '\n'},
																									basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, false, true, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
																									ignoreCase:      false,
																									inverted:        false,
																								},
																							},
																						},
																					},
																				},
																			},
																			&labeledExpr{
																				pos:   position{line: 225, col: 11, offset: 6364},
																				label: "indent",
																				expr: &zeroOrMoreExpr{
																					pos: position{line: 225, col: 18, offset: 6371},
																					expr: &charClassMatcher{
																						pos:             position{line: 232, col: 16, offset: 6493},
																						val:             "[ \\t]",
																						chars:           []rune{' ', '\t'},
																						basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
																						ignoreCase:      false,
																						inverted:        false,
																					},
																				},
																			},
																			&stateCodeExpr{
																				pos: position{line: 226, col: 3, offset: 6386},
																				run: (*parser).callonTestInternalBlocks177,
																			},
																			&labeledExpr{
																				pos:   position{line: 99, col: 18, offset: 2675},
																				label: "bullet",
																				expr: &actionExpr{
																					pos: position{line: 108, col: 15, offset: 2948},
																					run: (*parser).callonTestInternalBlocks179,
																					expr: &labeledExpr{
																						pos:   position{line: 108, col: 15, offset: 2948},
																						label: "bullet",
																						expr: &choiceExpr{
																							pos: position{line: 108, col: 24, offset: 2957},
																							alternatives: []interface{}{
																								&actionExpr{
																									pos: position{line: 113, col: 22, offset: 3052},
																									run: (*parser).callonTestInternalBlocks182,
																									expr: &oneOrMoreExpr{
																										pos: position{line: 113, col: 22, offset: 3052},
																										expr: &seqExpr{
																											pos: position{line: 113, col: 24, offset: 3054},
																											exprs: []interface{}{
																												&oneOrMoreExpr{
																													pos: position{line: 219, col: 11, offset: 6226},
																													expr: &charClassMatcher{
																														pos:             position{line: 218, col: 10, offset: 6210},
																														val:             "[0-9]",
																														ranges:          []rune{'0', '9'},
																														basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, true, true, true, true, true, true, true, true, true, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
																														ignoreCase:      false,
																														inverted:        false,
																													},
																												},
																												&litMatcher{
																													pos:        position{line: 220, col: 8, offset: 6240},
																													val:        ".",
																													ignoreCase: false,
																												},
																											},
																										},
																									},
																								},
																								&actionExpr{
																									pos: position{line: 118, col: 24, offset: 3131},
																									run: (*parser).callonTestInternalBlocks188,
																									expr: &litMatcher{
																										pos:        position{line: 118, col: 24, offset: 3131},
																										val:        "-",
																										ignoreCase: false,
																									},
																								},
																							},
																						},
																					},
																				},
																			},
																			&labeledExpr{
																				pos:   position{line: 99, col: 36, offset: 2693},
																				label: "text",
																				expr: &zeroOrOneExpr{
																					pos: position{line: 99, col: 41, offset: 2698},
																					expr: &seqExpr{
																						pos: position{line: 99, col: 43, offset: 2700},
																						exprs: []interface{}{
																							&charClassMatcher{
																								pos:             position{line: 233, col: 15, offset: 6513},
																								val:             "[ \\t\\f]",
																								chars:           []rune{' ', '\t', '\f'},
																								basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, true, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
																								ignoreCase:      false,
																								inverted:        false,
																							},
																							&zeroOrMoreExpr{
																								pos: position{line: 99, col: 54, offset: 2711},
																								expr: &seqExpr{
																									pos: position{line: 99, col: 56, offset: 2713},
																									exprs: []interface{}{
																										&notExpr{
																											pos: position{line: 99, col: 56, offset: 2713},
																											expr: &choiceExpr{
																												pos: position{line: 230, col: 10, offset: 6451},
																												alternatives: []interface{}{
																													&litMatcher{
																														pos:        position{line: 230, col: 10, offset: 6451},
																														val:        "\r\n\n\n\n\n\n\n\n\n\n",
																														ignoreCase: false,
																													},
																													&charClassMatcher{
																														pos:             position{line: 230, col: 22, offset: 6463},
																														val:             "[\\r\\n]",
																														chars:           []rune{'\r', '\n'},
																														basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, false, true, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
																														ignoreCase:      false,
																														inverted:        false,
																													},
																												},
																											},
																										},
																										&anyMatcher{
																											line: 99, col: 61, offset: 2718,
																										},
																									},
																								},
																							},
																						},
																					},
																				},
																			},
																			&choiceExpr{
																				pos: position{line: 230, col: 10, offset: 6451},
																				alternatives: []interface{}{
																					&litMatcher{
																						pos:        position{line: 230, col: 10, offset: 6451},
																						val:        "\r\n\n\n\n\n\n\n\n\n\n",
																						ignoreCase: false,
																					},
																					&charClassMatcher{
																						pos:             position{line: 230, col: 22, offset: 6463},
																						val:             "[\\r\\n]",
																						chars:           []rune{'\r', '\n'},
																						basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, false, true, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
																						ignoreCase:      false,
																						inverted:        false,
																					},
																				},
																			},
																			&labeledExpr{
																				pos:   position{line: 99, col: 73, offset: 2730},
																				label: "blank",
																				expr: &zeroOrOneExpr{
																					pos: position{line: 99, col: 79, offset: 2736},
																					expr: &actionExpr{
																						pos: position{line: 99, col: 81, offset: 2738},
																						run: (*parser).callonTestInternalBlocks206,
																						expr: &andExpr{
																							pos: position{line: 99, col: 81, offset: 2738},
																							expr: &seqExpr{
																								pos: position{line: 234, col: 14, offset: 6534},
																								exprs: []interface{}{
																									&zeroOrMoreExpr{
																										pos: position{line: 234, col: 14, offset: 6534},
																										expr: &charClassMatcher{
																											pos:             position{line: 233, col: 15, offset: 6513},
																											val:             "[ \\t\\f]",
																											chars:           []rune{' ', '\t', '\f'},
																											basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, true, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
																											ignoreCase:      false,
																											inverted:        false,
																										},
																									},
																									&choiceExpr{
																										pos: position{line: 230, col: 10, offset: 6451},
																										alternatives: []interface{}{
																											&litMatcher{
																												pos:        position{line: 230, col: 10, offset: 6451},
																												val:        "\r\n\n\n\n\n\n\n\n\n\n",
																												ignoreCase: false,
																											},
																											&charClassMatcher{
																												pos:             position{line: 230, col: 22, offset: 6463},
																												val:             "[\\r\\n]",
																												chars:           []rune{'\r', '\n'},
																												basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, false, true, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
																												ignoreCase:      false,
																												inverted:        false,
																											},
																										},
																									},
																								},
																							},
																						},
																					},
																				},
																			},
																		},
																	},
																},
															},
															&stateCodeExpr{
																pos: position{line: 40, col: 19, offset: 1038},
																run: (*parser).callonTestInternalBlocks214,
															},
														},
													},
												},
												&actionExpr{
													pos: position{line: 41, col: 7, offset: 1101},
													run: (*parser).callonTestInternalBlocks215,
													expr: &seqExpr{
														pos: position{line: 41, col: 7, offset: 1101},
														exprs: []interface{}{
															&labeledExpr{
																pos:   position{line: 41, col: 7, offset: 1101},
																label: "f",
																expr: &actionExpr{
																	pos: position{line: 131, col: 10, offset: 3545},
																	run: (*parser).callonTestInternalBlocks218,
																	expr: &seqExpr{
																		pos: position{line: 131, col: 10, offset: 3545},
																		exprs: []interface{}{
																			&zeroOrMoreExpr{
																				pos: position{line: 235, col: 6, offset: 6555},
																				expr: &seqExpr{
																					pos: position{line: 234, col: 14, offset: 6534},
																					exprs: []interface{}{
																						&zeroOrMoreExpr{
																							pos: position{line: 234, col: 14, offset: 6534},
																							expr: &charClassMatcher{
																								pos:             position{line: 233, col: 15, offset: 6513},
																								val:             "[ \\t\\f]",
																								chars:           []rune{' ', '\t', '\f'},
																								basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, true, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
																								ignoreCase:      false,
																								inverted:        false,
																							},
																						},
																						&choiceExpr{
																							pos: position{line: 230, col: 10, offset: 6451},
																							alternatives: []interface{}{
																								&litMatcher{
																									pos:        position{line: 230, col: 10, offset: 6451},
																									val:        "\r\n\n\n\n\n\n\n\n\n\n",
																									ignoreCase: false,
																								},
																								&charClassMatcher{
																									pos:             position{line: 230, col: 22, offset: 6463},
																									val:             "[\\r\\n]",
																									chars:           []rune{'\r', '\n'},
																									basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, false, true, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
																									ignoreCase:      false,
																									inverted:        false,
																								},
																							},
																						},
																					},
																				},
																			},
																			&labeledExpr{
																				pos:   position{line: 225, col: 11, offset: 6364},
																				label: "indent",
																				expr: &zeroOrMoreExpr{
																					pos: position{line: 225, col: 18, offset: 6371},
																					expr: &charClassMatcher{
																						pos:             position{line: 232, col: 16, offset: 6493},
																						val:             "[ \\t]",
																						chars:           []rune{' ', '\t'},
																						basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
																						ignoreCase:      false,
																						inverted:        false,
																					},
																				},
																			},
																			&stateCodeExpr{
																				pos: position{line: 226, col: 3, offset: 6386},
																				run: (*parser).callonTestInternalBlocks230,
																			},
																			&labeledExpr{
																				pos:   position{line: 131, col: 19, offset: 3554},
																				label: "tag",
																				expr: &actionExpr{
																					pos: position{line: 148, col: 13, offset: 4383},
																					run: (*parser).callonTestInternalBlocks232,
																					expr: &seqExpr{
																						pos: position{line: 148, col: 13, offset: 4383},
																						exprs: []interface{}{
																							&litMatcher{
																								pos:        position{line: 148, col: 13, offset: 4383},
																								val:        "@",
																								ignoreCase: false,
																							},
																							&labeledExpr{
																								pos:   position{line: 148, col: 17, offset: 4387},
																								label: "field",
																								expr: &actionExpr{
																									pos: position{line: 153, col: 15, offset: 4539},
																									run: (*parser).callonTestInternalBlocks236,
																									expr: &oneOrMoreExpr{
																										pos: position{line: 153, col: 15, offset: 4539},
																										expr: &charClassMatcher{
																											pos:             position{line: 153, col: 15, offset: 4539},
																											val:             "[^:\\pWhite_Space]",
																											chars:           []rune{':'},
																											classes:         []*unicode.RangeTable{rangeTable("White_Space")},
																											basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, true, true, true, true, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
																											ignoreCase:      false,
																											inverted:        true,
																										},
																									},
																								},
																							},
																							&labeledExpr{
																								pos:   position{line: 148, col: 34, offset: 4404},
																								label: "arg",
																								expr: &zeroOrOneExpr{
																									pos: position{line: 148, col: 38, offset: 4408},
																									expr: &seqExpr{
																										pos: position{line: 148, col: 40, offset: 4410},
																										exprs: []interface{}{
																											&oneOrMoreExpr{
																												pos: position{line: 148, col: 40, offset: 4410},
																												expr: &charClassMatcher{
																													pos:             position{line: 233, col: 15, offset: 6513},
																													val:             "[ \\t\\f]",
																													chars:           []rune{' ', '\t', '\f'},
																													basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, true, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
																													ignoreCase:      false,
																													inverted:        false,
																												},
																											},
																											&actionExpr{
																												pos: position{line: 153, col: 15, offset: 4539},
																												run: (*parser).callonTestInternalBlocks244,
																												expr: &oneOrMoreExpr{
																													pos: position{line: 153, col: 15, offset: 4539},
																													expr: &charClassMatcher{
																														pos:             position{line: 153, col: 15, offset: 4539},
																														val:             "[^:\\pWhite_Space]",
																														chars:           []rune{':'},
																														classes:         []*unicode.RangeTable{rangeTable("White_Space")},
																														basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, true, true, true, true, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
																														ignoreCase:      false,
																														inverted:        true,
																													},
																												},
																											},
																										},
																									},
																								},
																							},
																							&zeroOrMoreExpr{
																								pos: position{line: 148, col: 66, offset: 4436},
																								expr: &charClassMatcher{
																									pos:             position{line: 233, col: 15, offset: 6513},
																									val:             "[ \\t\\f]",
																									chars:           []rune{' ', '\t', '\f'},
																									basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, true, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
																									ignoreCase:      false,
																									inverted:        false,
																								},
																							},
																							&litMatcher{
																								pos:        position{line: 148, col: 78, offset: 4448},
																								val:        ":",
																								ignoreCase: false,
																							},
																						},
																					},
																				},
																			},
																			&labeledExpr{
																				pos:   position{line: 131, col: 32, offset: 3567},
																				label: "text",
																				expr: &zeroOrMoreExpr{
																					pos: position{line: 131, col: 37, offset: 3572},
																					expr: &seqExpr{
																						pos: position{line: 131, col: 39, offset: 3574},
																						exprs: []interface{}{
																							&notExpr{
																								pos: position{line: 131, col: 39, offset: 3574},
																								expr: &choiceExpr{
																									pos: position{line: 230, col: 10, offset: 6451},
																									alternatives: []interface{}{
																										&litMatcher{
																											pos:        position{line: 230, col: 10, offset: 6451},
																											val:        "\r\n\n\n\n\n\n\n\n\n\n",
																											ignoreCase: false,
																										},
																										&charClassMatcher{
																											pos:             position{line: 230, col: 22, offset: 6463},
																											val:             "[\\r\\n]",
																											chars:           []rune{'\r', '\n'},
																											basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, false, true, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
																											ignoreCase:      false,
																											inverted:        false,
																										},
																									},
																								},
																							},
																							&anyMatcher{
																								line: 131, col: 44, offset: 3579,
																							},
																						},
																					},
																				},
																			},
																			&choiceExpr{
																				pos: position{line: 230, col: 10, offset: 6451},
																				alternatives: []interface{}{
																					&litMatcher{
																						pos:        position{line: 230, col: 10, offset: 6451},
																						val:        "\r\n\n\n\n\n\n\n\n\n\n",
																						ignoreCase: false,
																					},
																					&charClassMatcher{
																						pos:             position{line: 230, col: 22, offset: 6463},
																						val:             "[\\r\\n]",
																						chars:           []rune{'\r', '\n'},
																						basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, false, true, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
																						ignoreCase:      false,
																						inverted:        false,
																					},
																				},
																			},
																			&labeledExpr{
																				pos:   position{line: 131, col: 53, offset: 3588},
																				label: "blank",
																				expr: &zeroOrOneExpr{
																					pos: position{line: 131, col: 59, offset: 3594},
																					expr: &actionExpr{
																						pos: position{line: 131, col: 61, offset: 3596},
																						run: (*parser).callonTestInternalBlocks263,
																						expr: &andExpr{
																							pos: position{line: 131, col: 61, offset: 3596},
																							expr: &seqExpr{
																								pos: position{line: 234, col: 14, offset: 6534},
																								exprs: []interface{}{
																									&zeroOrMoreExpr{
																										pos: position{line: 234, col: 14, offset: 6534},
																										expr: &charClassMatcher{
																											pos:             position{line: 233, col: 15, offset: 6513},
																											val:             "[ \\t\\f]",
																											chars:           []rune{' ', '\t', '\f'},
																											basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, true, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
																											ignoreCase:      false,
																											inverted:        false,
																										},
																									},
																									&choiceExpr{
																										pos: position{line: 230, col: 10, offset: 6451},
																										alternatives: []interface{}{
																											&litMatcher{
																												pos:        position{line: 230, col: 10, offset: 6451},
																												val:        "\r\n\n\n\n\n\n\n\n\n\n",
																												ignoreCase: false,
																											},
																											&charClassMatcher{
																												pos:             position{line: 230, col: 22, offset: 6463},
																												val:             "[\\r\\n]",
																												chars:           []rune{'\r', '\n'},
																												basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, false, true, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
																												ignoreCase:      false,
																												inverted:        false,
																											},
																										},
																									},
																								},
																							},
																						},
																					},
																				},
																			},
																		},
																	},
																},
															},
															&stateCodeExpr{
																pos: position{line: 41, col: 19, offset: 1113},
																run: (*parser).callonTestInternalBlocks271,
															},
														},
													},
												},
												&actionExpr{
													pos: position{line: 42, col: 7, offset: 1178},
													run: (*parser).callonTestInternalBlocks272,
													expr: &seqExpr{
														pos: position{line: 42, col: 7, offset: 1178},
														exprs: []interface{}{
															&labeledExpr{
																pos:   position{line: 42, col: 7, offset: 1178},
																label: "p",
																expr: &actionExpr{
																	pos: position{line: 163, col: 3, offset: 4806},
																	run: (*parser).callonTestInternalBlocks275,
																	expr: &seqExpr{
																		pos: position{line: 163, col: 3, offset: 4806},
																		exprs: []interface{}{
																			&zeroOrMoreExpr{
																				pos: position{line: 235, col: 6, offset: 6555},
																				expr: &seqExpr{
																					pos: position{line: 234, col: 14, offset: 6534},
																					exprs: []interface{}{
																						&zeroOrMoreExpr{
																							pos: position{line: 234, col: 14, offset: 6534},
																							expr: &charClassMatcher{
																								pos:             position{line: 233, col: 15, offset: 6513},
																								val:             "[ \\t\\f]",
																								chars:           []rune{' ', '\t', '\f'},
																								basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, true, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
																								ignoreCase:      false,
																								inverted:        false,
																							},
																						},
																						&choiceExpr{
																							pos: position{line: 230, col: 10, offset: 6451},
																							alternatives: []interface{}{
																								&litMatcher{
																									pos:        position{line: 230, col: 10, offset: 6451},
																									val:        "\r\n\n\n\n\n\n\n\n\n\n",
																									ignoreCase: false,
																								},
																								&charClassMatcher{
																									pos:             position{line: 230, col: 22, offset: 6463},
																									val:             "[\\r\\n]",
																									chars:           []rune{'\r', '\n'},
																									basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, false, true, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
																									ignoreCase:      false,
																									inverted:        false,
																								},
																							},
																						},
																					},
																				},
																			},
																			&labeledExpr{
																				pos:   position{line: 163, col: 5, offset: 4808},
																				label: "first",
																				expr: &actionExpr{
																					pos: position{line: 213, col: 17, offset: 6111},
																					run: (*parser).callonTestInternalBlocks285,
																					expr: &seqExpr{
																						pos: position{line: 213, col: 17, offset: 6111},
																						exprs: []interface{}{
																							&labeledExpr{
																								pos:   position{line: 225, col: 11, offset: 6364},
																								label: "indent",
																								expr: &zeroOrMoreExpr{
																									pos: position{line: 225, col: 18, offset: 6371},
																									expr: &charClassMatcher{
																										pos:             position{line: 232, col: 16, offset: 6493},
																										val:             "[ \\t]",
																										chars:           []rune{' ', '\t'},
																										basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
																										ignoreCase:      false,
																										inverted:        false,
																									},
																								},
																							},
																							&stateCodeExpr{
																								pos: position{line: 226, col: 3, offset: 6386},
																								run: (*parser).callonTestInternalBlocks290,
																							},
																							&labeledExpr{
																								pos:   position{line: 213, col: 24, offset: 6118},
																								label: "text",
																								expr: &oneOrMoreExpr{
																									pos: position{line: 213, col: 29, offset: 6123},
																									expr: &seqExpr{
																										pos: position{line: 213, col: 31, offset: 6125},
																										exprs: []interface{}{
																											&notExpr{
																												pos: position{line: 213, col: 31, offset: 6125},
																												expr: &choiceExpr{
																													pos: position{line: 230, col: 10, offset: 6451},
																													alternatives: []interface{}{
																														&litMatcher{
																															pos:        position{line: 230, col: 10, offset: 6451},
																															val:        "\r\n\n\n\n\n\n\n\n\n\n",
																															ignoreCase: false,
																														},
																														&charClassMatcher{
																															pos:             position{line: 230, col: 22, offset: 6463},
																															val:             "[\\r\\n]",
																															chars:           []rune{'\r', '\n'},
																															basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, false, true, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
																															ignoreCase:      false,
																															inverted:        false,
																														},
																													},
																												},
																											},
																											&anyMatcher{
																												line: 213, col: 36, offset: 6130,
																											},
																										},
																									},
																								},
																							},
																							&choiceExpr{
																								pos: position{line: 230, col: 10, offset: 6451},
																								alternatives: []interface{}{
																									&litMatcher{
																										pos:        position{line: 230, col: 10, offset: 6451},
																										val:        "\r\n\n\n\n\n\n\n\n\n\n",
																										ignoreCase: false,
																									},
																									&charClassMatcher{
																										pos:             position{line: 230, col: 22, offset: 6463},
																										val:             "[\\r\\n]",
																										chars:           []rune{'\r', '\n'},
																										basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, false, true, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
																										ignoreCase:      false,
																										inverted:        false,
																									},
																								},
																							},
																						},
																					},
																				},
																			},
																			&stateCodeExpr{
																				pos: position{line: 164, col: 3, offset: 4829},
																				run: (*parser).callonTestInternalBlocks302,
																			},
																			&labeledExpr{
																				pos:   position{line: 168, col: 3, offset: 4944},
																				label: "rest",
																				expr: &zeroOrMoreExpr{
																					pos: position{line: 168, col: 8, offset: 4949},
																					expr: &seqExpr{
																						pos: position{line: 169, col: 5, offset: 4955},
																						exprs: []interface{}{
																							&labeledExpr{
																								pos:   position{line: 169, col: 5, offset: 4955},
																								label: "line",
																								expr: &actionExpr{
																									pos: position{line: 213, col: 17, offset: 6111},
																									run: (*parser).callonTestInternalBlocks307,
																									expr: &seqExpr{
																										pos: position{line: 213, col: 17, offset: 6111},
																										exprs: []interface{}{
																											&labeledExpr{
																												pos:   position{line: 225, col: 11, offset: 6364},
																												label: "indent",
																												expr: &zeroOrMoreExpr{
																													pos: position{line: 225, col: 18, offset: 6371},
																													expr: &charClassMatcher{
																														pos:             position{line: 232, col: 16, offset: 6493},
																														val:             "[ \\t]",
																														chars:           []rune{' ', '\t'},
																														basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
																														ignoreCase:      false,
																														inverted:        false,
																													},
																												},
																											},
																											&stateCodeExpr{
																												pos: position{line: 226, col: 3, offset: 6386},
																												run: (*parser).callonTestInternalBlocks312,
																											},
																											&labeledExpr{
																												pos:   position{line: 213, col: 24, offset: 6118},
																												label: "text",
																												expr: &oneOrMoreExpr{
																													pos: position{line: 213, col: 29, offset: 6123},
																													expr: &seqExpr{
																														pos: position{line: 213, col: 31, offset: 6125},
																														exprs: []interface{}{
																															&notExpr{
																																pos: position{line: 213, col: 31, offset: 6125},
																																expr: &choiceExpr{
																																	pos: position{line: 230, col: 10, offset: 6451},
																																	alternatives: []interface{}{
																																		&litMatcher{
																																			pos:        position{line: 230, col: 10, offset: 6451},
																																			val:        "\r\n\n\n\n\n\n\n\n\n\n",
																																			ignoreCase: false,
																																		},
																																		&charClassMatcher{
																																			pos:             position{line: 230, col: 22, offset: 6463},
																																			val:             "[\\r\\n]",
																																			chars:           []rune{'\r', '\n'},
																																			basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, false, true, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
																																			ignoreCase:      false,
																																			inverted:        false,
																																		},
																																	},
																																},
																															},
																															&anyMatcher{
																																line: 213, col: 36, offset: 6130,
																															},
																														},
																													},
																												},
																											},
																											&choiceExpr{
																												pos: position{line: 230, col: 10, offset: 6451},
																												alternatives: []interface{}{
																													&litMatcher{
																														pos:        position{line: 230, col: 10, offset: 6451},
																														val:        "\r\n\n\n\n\n\n\n\n\n\n",
																														ignoreCase: false,
																													},
																													&charClassMatcher{
																														pos:             position{line: 230, col: 22, offset: 6463},
																														val:             "[\\r\\n]",
																														chars:           []rune{'\r', '\n'},
																														basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, false, true, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
																														ignoreCase:      false,
																														inverted:        false,
																													},
																												},
																											},
																										},
																									},
																								},
																							},
																							&andCodeExpr{
																								pos: position{line: 170, col: 5, offset: 4977},
																								run: (*parser).callonTestInternalBlocks324,
																							},
																						},
																					},
																				},
																			},
																		},
																	},
																},
															},
															&stateCodeExpr{
																pos: position{line: 42, col: 19, offset: 1190},
																run: (*parser).callonTestInternalBlocks325,
															},
														},
													},
												},
											},
										},
									},
								},
							},
						},
						&zeroOrMoreExpr{
							pos: position{line: 235, col: 6, offset: 6555},
							expr: &seqExpr{
								pos: position{line: 234, col: 14, offset: 6534},
								exprs: []interface{}{
									&zeroOrMoreExpr{
										pos: position{line: 234, col: 14, offset: 6534},
										expr: &charClassMatcher{
											pos:             position{line: 233, col: 15, offset: 6513},
											val:             "[ \\t\\f]",
											chars:           []rune{' ', '\t', '\f'},
											basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, true, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
											ignoreCase:      false,
											inverted:        false,
										},
									},
									&choiceExpr{
										pos: position{line: 230, col: 10, offset: 6451},
										alternatives: []interface{}{
											&litMatcher{
												pos:        position{line: 230, col: 10, offset: 6451},
												val:        "\r\n\n\n\n\n\n\n\n\n\n",
												ignoreCase: false,
											},
											&charClassMatcher{
												pos:             position{line: 230, col: 22, offset: 6463},
												val:             "[\\r\\n]",
												chars:           []rune{'\r', '\n'},
												basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, false, true, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
												ignoreCase:      false,
												inverted:        false,
											},
										},
									},
								},
							},
						},
						&zeroOrMoreExpr{
							pos: position{line: 27, col: 19, offset: 532},
							expr: &charClassMatcher{
								pos:             position{line: 233, col: 15, offset: 6513},
								val:             "[ \\t\\f]",
								chars:           []rune{' ', '\t', '\f'},
								basicLatinChars: [128]bool{false, false, false, false, false, false, false, false, false, true, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
								ignoreCase:      false,
								inverted:        false,
							},
						},
						&notExpr{
							pos: position{line: 237, col: 8, offset: 6574},
							expr: &anyMatcher{
								line: 237, col: 9, offset: 6575,
							},
						},
					},
				},
			},
		},
	},
}

func (c *current) onGrammar3() error {
	return initState(c)

}

func (p *parser) callonGrammar3() error {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onGrammar3()
}

func (c *current) onGrammar14() (bool, error) {
	// matches only if introduced by a paragraph that ends with "::"
	return literalIntroPredicate(c)

}

func (p *parser) callonGrammar14() (bool, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onGrammar14()
}

func (c *current) onGrammar31(indent interface{}) error {
	return indentState(c, toIfaceSlice(indent))

}

func (p *parser) callonGrammar31() error {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onGrammar31(stack["indent"])
}

func (c *current) onGrammar26(indent, text interface{}) (interface{}, error) {
	return nonBlankLineAction(c, toIfaceSlice(text))

}

func (p *parser) callonGrammar26() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onGrammar26(stack["indent"], stack["text"])
}

func (c *current) onGrammar43(line interface{}) (bool, error) {
	return literalLinePredicate(c, line.(plainText))

}

func (p *parser) callonGrammar43() (bool, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onGrammar43(stack["line"])
}

func (c *current) onGrammar12(lines interface{}) (interface{}, error) {
	return literalAction(c, toIfaceSlice(lines))

}

func (p *parser) callonGrammar12() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onGrammar12(stack["lines"])
}

func (c *current) onGrammar44(lit interface{}) error {
	return literalPostState(c, lit.(literal))
}

func (p *parser) callonGrammar44() error {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onGrammar44(stack["lit"])
}

func (c *current) onGrammar9(lit interface{}) (interface{}, error) {
	return lit, nil
}

func (p *parser) callonGrammar9() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onGrammar9(stack["lit"])
}

func (c *current) onGrammar60(indent interface{}) error {
	return indentState(c, toIfaceSlice(indent))

}

func (p *parser) callonGrammar60() error {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onGrammar60(stack["indent"])
}

func (c *current) onGrammar55(indent, text interface{}) (interface{}, error) {
	return nonBlankLineAction(c, toIfaceSlice(text))

}

func (p *parser) callonGrammar55() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onGrammar55(stack["indent"], stack["text"])
}

func (c *current) onGrammar78(indent interface{}) error {
	return indentState(c, toIfaceSlice(indent))

}

func (p *parser) callonGrammar78() error {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onGrammar78(stack["indent"])
}

func (c *current) onGrammar80() (interface{}, error) {
	return sectionUnderlineAction(c)

}

func (p *parser) callonGrammar80() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onGrammar80()
}

func (c *current) onGrammar73(indent, underline interface{}) (interface{}, error) {
	return underline, nil

}

func (p *parser) callonGrammar73() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onGrammar73(stack["indent"], stack["underline"])
}

func (c *current) onGrammar91(header, underline interface{}) (bool, error) {
	return sectionMatchPredicate(c, header.(plainText), underline.(plainText))

}

func (p *parser) callonGrammar91() (bool, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onGrammar91(stack["header"], stack["underline"])
}

func (c *current) onGrammar45(header, underline interface{}) (interface{}, error) {
	return sectionAction(c, header.(plainText), underline.(plainText))

}

func (p *parser) callonGrammar45() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onGrammar45(stack["header"], stack["underline"])
}

func (c *current) onGrammar113(indent interface{}) error {
	return indentState(c, toIfaceSlice(indent))

}

func (p *parser) callonGrammar113() error {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onGrammar113(stack["indent"])
}

func (c *current) onGrammar108(indent, text interface{}) (interface{}, error) {
	return firstDoctestLineAction(c, toIfaceSlice(text))

}

func (p *parser) callonGrammar108() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onGrammar108(stack["indent"], stack["text"])
}

func (c *current) onGrammar128(first interface{}) error {
	// store the current doctest's indentation
	return doctestFirstLineState(c, first.(plainText))

}

func (p *parser) callonGrammar128() error {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onGrammar128(stack["first"])
}

func (c *current) onGrammar138(indent interface{}) error {
	return indentState(c, toIfaceSlice(indent))

}

func (p *parser) callonGrammar138() error {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onGrammar138(stack["indent"])
}

func (c *current) onGrammar133(indent, text interface{}) (interface{}, error) {
	return nonBlankLineAction(c, toIfaceSlice(text))

}

func (p *parser) callonGrammar133() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onGrammar133(stack["indent"], stack["text"])
}

func (c *current) onGrammar150(line interface{}) (bool, error) {
	// matches only if the indentation of the line is the same as the
	// first line in the doctest.
	return doctestNextLinePredicate(c, line.(plainText))

}

func (p *parser) callonGrammar150() (bool, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onGrammar150(stack["line"])
}

func (c *current) onGrammar105(first, rest interface{}) (interface{}, error) {
	return doctestLinesAction(c, first.(plainText), toIfaceSlice(rest))

}

func (p *parser) callonGrammar105() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onGrammar105(stack["first"], stack["rest"])
}

func (c *current) onGrammar95(doctest interface{}) (interface{}, error) {
	return doctest, nil

}

func (p *parser) callonGrammar95() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onGrammar95(stack["doctest"])
}

func (c *current) onGrammar161(doc interface{}) error {
	return doctestPostState(c, doc.(doctest))
}

func (p *parser) callonGrammar161() error {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onGrammar161(stack["doc"])
}

func (c *current) onGrammar92(doc interface{}) (interface{}, error) {
	return doc, nil
}

func (p *parser) callonGrammar92() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onGrammar92(stack["doc"])
}

func (c *current) onGrammar177(indent interface{}) error {
	return indentState(c, toIfaceSlice(indent))

}

func (p *parser) callonGrammar177() error {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onGrammar177(stack["indent"])
}

func (c *current) onGrammar182() (interface{}, error) {
	return string(c.text), nil

}

func (p *parser) callonGrammar182() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onGrammar182()
}

func (c *current) onGrammar188() (interface{}, error) {
	return string(c.text), nil

}

func (p *parser) callonGrammar188() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onGrammar188()
}

func (c *current) onGrammar179(bullet interface{}) (interface{}, error) {
	return bullet, nil

}

func (p *parser) callonGrammar179() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onGrammar179(stack["bullet"])
}

func (c *current) onGrammar206() (interface{}, error) {
	return true, nil
}

func (p *parser) callonGrammar206() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onGrammar206()
}

func (c *current) onGrammar165(indent, bullet, text, blank interface{}) (interface{}, error) {
	var hasBlank bool
	if blank != nil {
		hasBlank = blank.(bool)
	}
	return listAction(c, bullet.(string), toIfaceSlice(text), hasBlank)

}

func (p *parser) callonGrammar165() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onGrammar165(stack["indent"], stack["bullet"], stack["text"], stack["blank"])
}

func (c *current) onGrammar214(l interface{}) error {
	return listPostState(c, l.(list))
}

func (p *parser) callonGrammar214() error {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onGrammar214(stack["l"])
}

func (c *current) onGrammar162(l interface{}) (interface{}, error) {
	return l, nil
}

func (p *parser) callonGrammar162() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onGrammar162(stack["l"])
}

func (c *current) onGrammar230(indent interface{}) error {
	return indentState(c, toIfaceSlice(indent))

}

func (p *parser) callonGrammar230() error {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onGrammar230(stack["indent"])
}

func (c *current) onGrammar236() (interface{}, error) {
	return string(c.text), nil

}

func (p *parser) callonGrammar236() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onGrammar236()
}

func (c *current) onGrammar244() (interface{}, error) {
	return string(c.text), nil

}

func (p *parser) callonGrammar244() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onGrammar244()
}

func (c *current) onGrammar232(field, arg interface{}) (interface{}, error) {
	return fieldTagAction(c, field.(string), toIfaceSlice(arg))

}

func (p *parser) callonGrammar232() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onGrammar232(stack["field"], stack["arg"])
}

func (c *current) onGrammar263() (interface{}, error) {
	return true, nil
}

func (p *parser) callonGrammar263() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onGrammar263()
}

func (c *current) onGrammar218(indent, tag, text, blank interface{}) (interface{}, error) {
	var hasBlank bool
	if blank != nil {
		hasBlank = blank.(bool)
	}
	return fieldAction(c, tag.(fieldTag), toIfaceSlice(text), hasBlank)

}

func (p *parser) callonGrammar218() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onGrammar218(stack["indent"], stack["tag"], stack["text"], stack["blank"])
}

func (c *current) onGrammar271(f interface{}) error {
	return fieldPostState(c, f.(field))
}

func (p *parser) callonGrammar271() error {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onGrammar271(stack["f"])
}

func (c *current) onGrammar215(f interface{}) (interface{}, error) {
	return f, nil
}

func (p *parser) callonGrammar215() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onGrammar215(stack["f"])
}

func (c *current) onGrammar290(indent interface{}) error {
	return indentState(c, toIfaceSlice(indent))

}

func (p *parser) callonGrammar290() error {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onGrammar290(stack["indent"])
}

func (c *current) onGrammar285(indent, text interface{}) (interface{}, error) {
	return nonBlankLineAction(c, toIfaceSlice(text))

}

func (p *parser) callonGrammar285() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onGrammar285(stack["indent"], stack["text"])
}

func (c *current) onGrammar302(first interface{}) error {
	// store the current paragraph's indentation
	return paragraphFirstLineState(c, first.(plainText))

}

func (p *parser) callonGrammar302() error {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onGrammar302(stack["first"])
}

func (c *current) onGrammar312(indent interface{}) error {
	return indentState(c, toIfaceSlice(indent))

}

func (p *parser) callonGrammar312() error {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onGrammar312(stack["indent"])
}

func (c *current) onGrammar307(indent, text interface{}) (interface{}, error) {
	return nonBlankLineAction(c, toIfaceSlice(text))

}

func (p *parser) callonGrammar307() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onGrammar307(stack["indent"], stack["text"])
}

func (c *current) onGrammar324(line interface{}) (bool, error) {
	// matches only if the indentation of the line is the same as the
	// first line in the paragraph.
	return paragraphNextLinePredicate(c, line.(plainText))

}

func (p *parser) callonGrammar324() (bool, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onGrammar324(stack["line"])
}

func (c *current) onGrammar275(first, rest interface{}) (interface{}, error) {
	return paragraphAction(c, first.(plainText), toIfaceSlice(rest))

}

func (p *parser) callonGrammar275() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onGrammar275(stack["first"], stack["rest"])
}

func (c *current) onGrammar325(p interface{}) error {
	return paragraphPostState(c, p.(paragraph))
}

func (p *parser) callonGrammar325() error {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onGrammar325(stack["p"])
}

func (c *current) onGrammar272(p interface{}) (interface{}, error) {
	return p, nil
}

func (p *parser) callonGrammar272() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onGrammar272(stack["p"])
}

func (c *current) onGrammar6(block interface{}) (interface{}, error) {
	return block, nil

}

func (p *parser) callonGrammar6() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onGrammar6(stack["block"])
}

func (c *current) onGrammar1(blocks interface{}) (interface{}, error) {
	return grammarAction(c, toIfaceSlice(blocks))

}

func (p *parser) callonGrammar1() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onGrammar1(stack["blocks"])
}

func (c *current) onTestInternalBlocks3() error {
	return initState(c)

}

func (p *parser) callonTestInternalBlocks3() error {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onTestInternalBlocks3()
}

func (c *current) onTestInternalBlocks14() (bool, error) {
	// matches only if introduced by a paragraph that ends with "::"
	return literalIntroPredicate(c)

}

func (p *parser) callonTestInternalBlocks14() (bool, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onTestInternalBlocks14()
}

func (c *current) onTestInternalBlocks31(indent interface{}) error {
	return indentState(c, toIfaceSlice(indent))

}

func (p *parser) callonTestInternalBlocks31() error {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onTestInternalBlocks31(stack["indent"])
}

func (c *current) onTestInternalBlocks26(indent, text interface{}) (interface{}, error) {
	return nonBlankLineAction(c, toIfaceSlice(text))

}

func (p *parser) callonTestInternalBlocks26() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onTestInternalBlocks26(stack["indent"], stack["text"])
}

func (c *current) onTestInternalBlocks43(line interface{}) (bool, error) {
	return literalLinePredicate(c, line.(plainText))

}

func (p *parser) callonTestInternalBlocks43() (bool, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onTestInternalBlocks43(stack["line"])
}

func (c *current) onTestInternalBlocks12(lines interface{}) (interface{}, error) {
	return literalAction(c, toIfaceSlice(lines))

}

func (p *parser) callonTestInternalBlocks12() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onTestInternalBlocks12(stack["lines"])
}

func (c *current) onTestInternalBlocks44(lit interface{}) error {
	return literalPostState(c, lit.(literal))
}

func (p *parser) callonTestInternalBlocks44() error {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onTestInternalBlocks44(stack["lit"])
}

func (c *current) onTestInternalBlocks9(lit interface{}) (interface{}, error) {
	return lit, nil
}

func (p *parser) callonTestInternalBlocks9() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onTestInternalBlocks9(stack["lit"])
}

func (c *current) onTestInternalBlocks60(indent interface{}) error {
	return indentState(c, toIfaceSlice(indent))

}

func (p *parser) callonTestInternalBlocks60() error {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onTestInternalBlocks60(stack["indent"])
}

func (c *current) onTestInternalBlocks55(indent, text interface{}) (interface{}, error) {
	return nonBlankLineAction(c, toIfaceSlice(text))

}

func (p *parser) callonTestInternalBlocks55() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onTestInternalBlocks55(stack["indent"], stack["text"])
}

func (c *current) onTestInternalBlocks78(indent interface{}) error {
	return indentState(c, toIfaceSlice(indent))

}

func (p *parser) callonTestInternalBlocks78() error {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onTestInternalBlocks78(stack["indent"])
}

func (c *current) onTestInternalBlocks80() (interface{}, error) {
	return sectionUnderlineAction(c)

}

func (p *parser) callonTestInternalBlocks80() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onTestInternalBlocks80()
}

func (c *current) onTestInternalBlocks73(indent, underline interface{}) (interface{}, error) {
	return underline, nil

}

func (p *parser) callonTestInternalBlocks73() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onTestInternalBlocks73(stack["indent"], stack["underline"])
}

func (c *current) onTestInternalBlocks91(header, underline interface{}) (bool, error) {
	return sectionMatchPredicate(c, header.(plainText), underline.(plainText))

}

func (p *parser) callonTestInternalBlocks91() (bool, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onTestInternalBlocks91(stack["header"], stack["underline"])
}

func (c *current) onTestInternalBlocks45(header, underline interface{}) (interface{}, error) {
	return sectionAction(c, header.(plainText), underline.(plainText))

}

func (p *parser) callonTestInternalBlocks45() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onTestInternalBlocks45(stack["header"], stack["underline"])
}

func (c *current) onTestInternalBlocks113(indent interface{}) error {
	return indentState(c, toIfaceSlice(indent))

}

func (p *parser) callonTestInternalBlocks113() error {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onTestInternalBlocks113(stack["indent"])
}

func (c *current) onTestInternalBlocks108(indent, text interface{}) (interface{}, error) {
	return firstDoctestLineAction(c, toIfaceSlice(text))

}

func (p *parser) callonTestInternalBlocks108() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onTestInternalBlocks108(stack["indent"], stack["text"])
}

func (c *current) onTestInternalBlocks128(first interface{}) error {
	// store the current doctest's indentation
	return doctestFirstLineState(c, first.(plainText))

}

func (p *parser) callonTestInternalBlocks128() error {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onTestInternalBlocks128(stack["first"])
}

func (c *current) onTestInternalBlocks138(indent interface{}) error {
	return indentState(c, toIfaceSlice(indent))

}

func (p *parser) callonTestInternalBlocks138() error {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onTestInternalBlocks138(stack["indent"])
}

func (c *current) onTestInternalBlocks133(indent, text interface{}) (interface{}, error) {
	return nonBlankLineAction(c, toIfaceSlice(text))

}

func (p *parser) callonTestInternalBlocks133() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onTestInternalBlocks133(stack["indent"], stack["text"])
}

func (c *current) onTestInternalBlocks150(line interface{}) (bool, error) {
	// matches only if the indentation of the line is the same as the
	// first line in the doctest.
	return doctestNextLinePredicate(c, line.(plainText))

}

func (p *parser) callonTestInternalBlocks150() (bool, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onTestInternalBlocks150(stack["line"])
}

func (c *current) onTestInternalBlocks105(first, rest interface{}) (interface{}, error) {
	return doctestLinesAction(c, first.(plainText), toIfaceSlice(rest))

}

func (p *parser) callonTestInternalBlocks105() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onTestInternalBlocks105(stack["first"], stack["rest"])
}

func (c *current) onTestInternalBlocks95(doctest interface{}) (interface{}, error) {
	return doctest, nil

}

func (p *parser) callonTestInternalBlocks95() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onTestInternalBlocks95(stack["doctest"])
}

func (c *current) onTestInternalBlocks161(doc interface{}) error {
	return doctestPostState(c, doc.(doctest))
}

func (p *parser) callonTestInternalBlocks161() error {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onTestInternalBlocks161(stack["doc"])
}

func (c *current) onTestInternalBlocks92(doc interface{}) (interface{}, error) {
	return doc, nil
}

func (p *parser) callonTestInternalBlocks92() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onTestInternalBlocks92(stack["doc"])
}

func (c *current) onTestInternalBlocks177(indent interface{}) error {
	return indentState(c, toIfaceSlice(indent))

}

func (p *parser) callonTestInternalBlocks177() error {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onTestInternalBlocks177(stack["indent"])
}

func (c *current) onTestInternalBlocks182() (interface{}, error) {
	return string(c.text), nil

}

func (p *parser) callonTestInternalBlocks182() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onTestInternalBlocks182()
}

func (c *current) onTestInternalBlocks188() (interface{}, error) {
	return string(c.text), nil

}

func (p *parser) callonTestInternalBlocks188() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onTestInternalBlocks188()
}

func (c *current) onTestInternalBlocks179(bullet interface{}) (interface{}, error) {
	return bullet, nil

}

func (p *parser) callonTestInternalBlocks179() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onTestInternalBlocks179(stack["bullet"])
}

func (c *current) onTestInternalBlocks206() (interface{}, error) {
	return true, nil
}

func (p *parser) callonTestInternalBlocks206() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onTestInternalBlocks206()
}

func (c *current) onTestInternalBlocks165(indent, bullet, text, blank interface{}) (interface{}, error) {
	var hasBlank bool
	if blank != nil {
		hasBlank = blank.(bool)
	}
	return listAction(c, bullet.(string), toIfaceSlice(text), hasBlank)

}

func (p *parser) callonTestInternalBlocks165() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onTestInternalBlocks165(stack["indent"], stack["bullet"], stack["text"], stack["blank"])
}

func (c *current) onTestInternalBlocks214(l interface{}) error {
	return listPostState(c, l.(list))
}

func (p *parser) callonTestInternalBlocks214() error {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onTestInternalBlocks214(stack["l"])
}

func (c *current) onTestInternalBlocks162(l interface{}) (interface{}, error) {
	return l, nil
}

func (p *parser) callonTestInternalBlocks162() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onTestInternalBlocks162(stack["l"])
}

func (c *current) onTestInternalBlocks230(indent interface{}) error {
	return indentState(c, toIfaceSlice(indent))

}

func (p *parser) callonTestInternalBlocks230() error {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onTestInternalBlocks230(stack["indent"])
}

func (c *current) onTestInternalBlocks236() (interface{}, error) {
	return string(c.text), nil

}

func (p *parser) callonTestInternalBlocks236() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onTestInternalBlocks236()
}

func (c *current) onTestInternalBlocks244() (interface{}, error) {
	return string(c.text), nil

}

func (p *parser) callonTestInternalBlocks244() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onTestInternalBlocks244()
}

func (c *current) onTestInternalBlocks232(field, arg interface{}) (interface{}, error) {
	return fieldTagAction(c, field.(string), toIfaceSlice(arg))

}

func (p *parser) callonTestInternalBlocks232() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onTestInternalBlocks232(stack["field"], stack["arg"])
}

func (c *current) onTestInternalBlocks263() (interface{}, error) {
	return true, nil
}

func (p *parser) callonTestInternalBlocks263() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onTestInternalBlocks263()
}

func (c *current) onTestInternalBlocks218(indent, tag, text, blank interface{}) (interface{}, error) {
	var hasBlank bool
	if blank != nil {
		hasBlank = blank.(bool)
	}
	return fieldAction(c, tag.(fieldTag), toIfaceSlice(text), hasBlank)

}

func (p *parser) callonTestInternalBlocks218() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onTestInternalBlocks218(stack["indent"], stack["tag"], stack["text"], stack["blank"])
}

func (c *current) onTestInternalBlocks271(f interface{}) error {
	return fieldPostState(c, f.(field))
}

func (p *parser) callonTestInternalBlocks271() error {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onTestInternalBlocks271(stack["f"])
}

func (c *current) onTestInternalBlocks215(f interface{}) (interface{}, error) {
	return f, nil
}

func (p *parser) callonTestInternalBlocks215() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onTestInternalBlocks215(stack["f"])
}

func (c *current) onTestInternalBlocks290(indent interface{}) error {
	return indentState(c, toIfaceSlice(indent))

}

func (p *parser) callonTestInternalBlocks290() error {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onTestInternalBlocks290(stack["indent"])
}

func (c *current) onTestInternalBlocks285(indent, text interface{}) (interface{}, error) {
	return nonBlankLineAction(c, toIfaceSlice(text))

}

func (p *parser) callonTestInternalBlocks285() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onTestInternalBlocks285(stack["indent"], stack["text"])
}

func (c *current) onTestInternalBlocks302(first interface{}) error {
	// store the current paragraph's indentation
	return paragraphFirstLineState(c, first.(plainText))

}

func (p *parser) callonTestInternalBlocks302() error {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onTestInternalBlocks302(stack["first"])
}

func (c *current) onTestInternalBlocks312(indent interface{}) error {
	return indentState(c, toIfaceSlice(indent))

}

func (p *parser) callonTestInternalBlocks312() error {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onTestInternalBlocks312(stack["indent"])
}

func (c *current) onTestInternalBlocks307(indent, text interface{}) (interface{}, error) {
	return nonBlankLineAction(c, toIfaceSlice(text))

}

func (p *parser) callonTestInternalBlocks307() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onTestInternalBlocks307(stack["indent"], stack["text"])
}

func (c *current) onTestInternalBlocks324(line interface{}) (bool, error) {
	// matches only if the indentation of the line is the same as the
	// first line in the paragraph.
	return paragraphNextLinePredicate(c, line.(plainText))

}

func (p *parser) callonTestInternalBlocks324() (bool, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onTestInternalBlocks324(stack["line"])
}

func (c *current) onTestInternalBlocks275(first, rest interface{}) (interface{}, error) {
	return paragraphAction(c, first.(plainText), toIfaceSlice(rest))

}

func (p *parser) callonTestInternalBlocks275() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onTestInternalBlocks275(stack["first"], stack["rest"])
}

func (c *current) onTestInternalBlocks325(p interface{}) error {
	return paragraphPostState(c, p.(paragraph))
}

func (p *parser) callonTestInternalBlocks325() error {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onTestInternalBlocks325(stack["p"])
}

func (c *current) onTestInternalBlocks272(p interface{}) (interface{}, error) {
	return p, nil
}

func (p *parser) callonTestInternalBlocks272() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onTestInternalBlocks272(stack["p"])
}

func (c *current) onTestInternalBlocks6(block interface{}) (interface{}, error) {
	return block, nil

}

func (p *parser) callonTestInternalBlocks6() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onTestInternalBlocks6(stack["block"])
}

func (c *current) onTestInternalBlocks1(blocks interface{}) (interface{}, error) {
	return blocks, nil

}

func (p *parser) callonTestInternalBlocks1() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onTestInternalBlocks1(stack["blocks"])
}

var (
	// errNoRule is returned when the grammar to parse has no rule.
	errNoRule = errors.New("grammar has no rule")

	// errInvalidEntrypoint is returned when the specified entrypoint rule
	// does not exit.
	errInvalidEntrypoint = errors.New("invalid entrypoint")

	// errInvalidEncoding is returned when the source is not properly
	// utf8-encoded.
	errInvalidEncoding = errors.New("invalid encoding")

	// errMaxExprCnt is used to signal that the maximum number of
	// expressions have been parsed.
	errMaxExprCnt = errors.New("max number of expresssions parsed")
)

// Option is a function that can set an option on the parser. It returns
// the previous setting as an Option.
type Option func(*parser) Option

// MaxExpressions creates an Option to stop parsing after the provided
// number of expressions have been parsed, if the value is 0 then the parser will
// parse for as many steps as needed (possibly an infinite number).
//
// The default for maxExprCnt is 0.
func MaxExpressions(maxExprCnt uint64) Option {
	return func(p *parser) Option {
		oldMaxExprCnt := p.maxExprCnt
		p.maxExprCnt = maxExprCnt
		return MaxExpressions(oldMaxExprCnt)
	}
}

// Entrypoint creates an Option to set the rule name to use as entrypoint.
// The rule name must have been specified in the -alternate-entrypoints
// if generating the parser with the -optimize-grammar flag, otherwise
// it may have been optimized out. Passing an empty string sets the
// entrypoint to the first rule in the grammar.
//
// The default is to start parsing at the first rule in the grammar.
func Entrypoint(ruleName string) Option {
	return func(p *parser) Option {
		oldEntrypoint := p.entrypoint
		p.entrypoint = ruleName
		if ruleName == "" {
			p.entrypoint = g.rules[0].name
		}
		return Entrypoint(oldEntrypoint)
	}
}

// AllowInvalidUTF8 creates an Option to allow invalid UTF-8 bytes.
// Every invalid UTF-8 byte is treated as a utf8.RuneError (U+FFFD)
// by character class matchers and is matched by the any matcher.
// The returned matched value, c.text and c.offset are NOT affected.
//
// The default is false.
func AllowInvalidUTF8(b bool) Option {
	return func(p *parser) Option {
		old := p.allowInvalidUTF8
		p.allowInvalidUTF8 = b
		return AllowInvalidUTF8(old)
	}
}

// Recover creates an Option to set the recover flag to b. When set to
// true, this causes the parser to recover from panics and convert it
// to an error. Setting it to false can be useful while debugging to
// access the full stack trace.
//
// The default is true.
func Recover(b bool) Option {
	return func(p *parser) Option {
		old := p.recover
		p.recover = b
		return Recover(old)
	}
}

// GlobalStore creates an Option to set a key to a certain value in
// the globalStore.
func GlobalStore(key string, value interface{}) Option {
	return func(p *parser) Option {
		old := p.cur.globalStore[key]
		p.cur.globalStore[key] = value
		return GlobalStore(key, old)
	}
}

// InitState creates an Option to set a key to a certain value in
// the global "state" store.
func InitState(key string, value interface{}) Option {
	return func(p *parser) Option {
		old := p.cur.state[key]
		p.cur.state[key] = value
		return InitState(key, old)
	}
}

// ParseFile parses the file identified by filename.
func ParseFile(filename string, opts ...Option) (i interface{}, err error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil {
			err = closeErr
		}
	}()
	return ParseReader(filename, f, opts...)
}

// ParseReader parses the data from r using filename as information in the
// error messages.
func ParseReader(filename string, r io.Reader, opts ...Option) (interface{}, error) {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	return Parse(filename, b, opts...)
}

// Parse parses the data from b using filename as information in the
// error messages.
func Parse(filename string, b []byte, opts ...Option) (interface{}, error) {
	return newParser(filename, b, opts...).parse(g)
}

// position records a position in the text.
type position struct {
	line, col, offset int
}

func (p position) String() string {
	return fmt.Sprintf("%d:%d [%d]", p.line, p.col, p.offset)
}

// savepoint stores all state required to go back to this point in the
// parser.
type savepoint struct {
	position
	rn rune
	w  int
}

type current struct {
	pos  position // start position of the match
	text []byte   // raw text of the match

	// state is a store for arbitrary key,value pairs that the user wants to be
	// tied to the backtracking of the parser.
	// This is always rolled back if a parsing rule fails.
	state storeDict

	// globalStore is a general store for the user to store arbitrary key-value
	// pairs that they need to manage and that they do not want tied to the
	// backtracking of the parser. This is only modified by the user and never
	// rolled back by the parser. It is always up to the user to keep this in a
	// consistent state.
	globalStore storeDict
}

type storeDict map[string]interface{}

// the AST types...

type grammar struct {
	pos   position
	rules []*rule
}

type rule struct {
	pos         position
	name        string
	displayName string
	expr        interface{}
}

type choiceExpr struct {
	pos          position
	alternatives []interface{}
}

type actionExpr struct {
	pos  position
	expr interface{}
	run  func(*parser) (interface{}, error)
}

type recoveryExpr struct {
	pos          position
	expr         interface{}
	recoverExpr  interface{}
	failureLabel []string
}

type seqExpr struct {
	pos   position
	exprs []interface{}
}

type throwExpr struct {
	pos   position
	label string
}

type labeledExpr struct {
	pos   position
	label string
	expr  interface{}
}

type expr struct {
	pos  position
	expr interface{}
}

type andExpr expr
type notExpr expr
type zeroOrOneExpr expr
type zeroOrMoreExpr expr
type oneOrMoreExpr expr

type ruleRefExpr struct {
	pos  position
	name string
}

type stateCodeExpr struct {
	pos position
	run func(*parser) error
}

type andCodeExpr struct {
	pos position
	run func(*parser) (bool, error)
}

type notCodeExpr struct {
	pos position
	run func(*parser) (bool, error)
}

type litMatcher struct {
	pos        position
	val        string
	ignoreCase bool
}

type charClassMatcher struct {
	pos             position
	val             string
	basicLatinChars [128]bool
	chars           []rune
	ranges          []rune
	classes         []*unicode.RangeTable
	ignoreCase      bool
	inverted        bool
}

type anyMatcher position

// errList cumulates the errors found by the parser.
type errList []error

func (e *errList) add(err error) {
	*e = append(*e, err)
}

func (e errList) err() error {
	if len(e) == 0 {
		return nil
	}
	e.dedupe()
	return e
}

func (e *errList) dedupe() {
	var cleaned []error
	set := make(map[string]bool)
	for _, err := range *e {
		if msg := err.Error(); !set[msg] {
			set[msg] = true
			cleaned = append(cleaned, err)
		}
	}
	*e = cleaned
}

func (e errList) Error() string {
	switch len(e) {
	case 0:
		return ""
	case 1:
		return e[0].Error()
	default:
		var buf bytes.Buffer

		for i, err := range e {
			if i > 0 {
				buf.WriteRune('\n')
			}
			buf.WriteString(err.Error())
		}
		return buf.String()
	}
}

// parserError wraps an error with a prefix indicating the rule in which
// the error occurred. The original error is stored in the Inner field.
type parserError struct {
	Inner    error
	pos      position
	prefix   string
	expected []string
}

// Error returns the error message.
func (p *parserError) Error() string {
	return p.prefix + ": " + p.Inner.Error()
}

// newParser creates a parser with the specified input source and options.
func newParser(filename string, b []byte, opts ...Option) *parser {
	stats := Stats{
		ChoiceAltCnt: make(map[string]map[string]int),
	}

	p := &parser{
		filename: filename,
		errs:     new(errList),
		data:     b,
		pt:       savepoint{position: position{line: 1}},
		recover:  true,
		cur: current{
			state:       make(storeDict),
			globalStore: make(storeDict),
		},
		maxFailPos:      position{col: 1, line: 1},
		maxFailExpected: make([]string, 0, 20),
		Stats:           &stats,
		// start rule is rule [0] unless an alternate entrypoint is specified
		entrypoint: g.rules[0].name,
		emptyState: make(storeDict),
	}
	p.setOptions(opts)

	if p.maxExprCnt == 0 {
		p.maxExprCnt = math.MaxUint64
	}

	return p
}

// setOptions applies the options to the parser.
func (p *parser) setOptions(opts []Option) {
	for _, opt := range opts {
		opt(p)
	}
}

type resultTuple struct {
	v   interface{}
	b   bool
	end savepoint
}

const choiceNoMatch = -1

// Stats stores some statistics, gathered during parsing
type Stats struct {
	// ExprCnt counts the number of expressions processed during parsing
	// This value is compared to the maximum number of expressions allowed
	// (set by the MaxExpressions option).
	ExprCnt uint64

	// ChoiceAltCnt is used to count for each ordered choice expression,
	// which alternative is used how may times.
	// These numbers allow to optimize the order of the ordered choice expression
	// to increase the performance of the parser
	//
	// The outer key of ChoiceAltCnt is composed of the name of the rule as well
	// as the line and the column of the ordered choice.
	// The inner key of ChoiceAltCnt is the number (one-based) of the matching alternative.
	// For each alternative the number of matches are counted. If an ordered choice does not
	// match, a special counter is incremented. The name of this counter is set with
	// the parser option Statistics.
	// For an alternative to be included in ChoiceAltCnt, it has to match at least once.
	ChoiceAltCnt map[string]map[string]int
}

type parser struct {
	filename string
	pt       savepoint
	cur      current

	data []byte
	errs *errList

	depth   int
	recover bool

	// rules table, maps the rule identifier to the rule node
	rules map[string]*rule
	// variables stack, map of label to value
	vstack []map[string]interface{}
	// rule stack, allows identification of the current rule in errors
	rstack []*rule

	// parse fail
	maxFailPos            position
	maxFailExpected       []string
	maxFailInvertExpected bool

	// max number of expressions to be parsed
	maxExprCnt uint64
	// entrypoint for the parser
	entrypoint string

	allowInvalidUTF8 bool

	*Stats

	choiceNoMatch string
	// recovery expression stack, keeps track of the currently available recovery expression, these are traversed in reverse
	recoveryStack []map[string]interface{}

	// emptyState contains an empty storeDict, which is used to optimize cloneState if global "state" store is not used.
	emptyState storeDict
}

// push a variable set on the vstack.
func (p *parser) pushV() {
	if cap(p.vstack) == len(p.vstack) {
		// create new empty slot in the stack
		p.vstack = append(p.vstack, nil)
	} else {
		// slice to 1 more
		p.vstack = p.vstack[:len(p.vstack)+1]
	}

	// get the last args set
	m := p.vstack[len(p.vstack)-1]
	if m != nil && len(m) == 0 {
		// empty map, all good
		return
	}

	m = make(map[string]interface{})
	p.vstack[len(p.vstack)-1] = m
}

// pop a variable set from the vstack.
func (p *parser) popV() {
	// if the map is not empty, clear it
	m := p.vstack[len(p.vstack)-1]
	if len(m) > 0 {
		// GC that map
		p.vstack[len(p.vstack)-1] = nil
	}
	p.vstack = p.vstack[:len(p.vstack)-1]
}

// push a recovery expression with its labels to the recoveryStack
func (p *parser) pushRecovery(labels []string, expr interface{}) {
	if cap(p.recoveryStack) == len(p.recoveryStack) {
		// create new empty slot in the stack
		p.recoveryStack = append(p.recoveryStack, nil)
	} else {
		// slice to 1 more
		p.recoveryStack = p.recoveryStack[:len(p.recoveryStack)+1]
	}

	m := make(map[string]interface{}, len(labels))
	for _, fl := range labels {
		m[fl] = expr
	}
	p.recoveryStack[len(p.recoveryStack)-1] = m
}

// pop a recovery expression from the recoveryStack
func (p *parser) popRecovery() {
	// GC that map
	p.recoveryStack[len(p.recoveryStack)-1] = nil

	p.recoveryStack = p.recoveryStack[:len(p.recoveryStack)-1]
}

func (p *parser) addErr(err error) {
	p.addErrAt(err, p.pt.position, []string{})
}

func (p *parser) addErrAt(err error, pos position, expected []string) {
	var buf bytes.Buffer
	if p.filename != "" {
		buf.WriteString(p.filename)
	}
	if buf.Len() > 0 {
		buf.WriteString(":")
	}
	buf.WriteString(fmt.Sprintf("%d:%d (%d)", pos.line, pos.col, pos.offset))
	if len(p.rstack) > 0 {
		if buf.Len() > 0 {
			buf.WriteString(": ")
		}
		rule := p.rstack[len(p.rstack)-1]
		if rule.displayName != "" {
			buf.WriteString("rule " + rule.displayName)
		} else {
			buf.WriteString("rule " + rule.name)
		}
	}
	pe := &parserError{Inner: err, pos: pos, prefix: buf.String(), expected: expected}
	p.errs.add(pe)
}

func (p *parser) failAt(fail bool, pos position, want string) {
	// process fail if parsing fails and not inverted or parsing succeeds and invert is set
	if fail == p.maxFailInvertExpected {
		if pos.offset < p.maxFailPos.offset {
			return
		}

		if pos.offset > p.maxFailPos.offset {
			p.maxFailPos = pos
			p.maxFailExpected = p.maxFailExpected[:0]
		}

		if p.maxFailInvertExpected {
			want = "!" + want
		}
		p.maxFailExpected = append(p.maxFailExpected, want)
	}
}

// read advances the parser to the next rune.
func (p *parser) read() {
	p.pt.offset += p.pt.w
	rn, n := utf8.DecodeRune(p.data[p.pt.offset:])
	p.pt.rn = rn
	p.pt.w = n
	p.pt.col++
	if rn == '\n' {
		p.pt.line++
		p.pt.col = 0
	}

	if rn == utf8.RuneError && n == 1 { // see utf8.DecodeRune
		if !p.allowInvalidUTF8 {
			p.addErr(errInvalidEncoding)
		}
	}
}

// restore parser position to the savepoint pt.
func (p *parser) restore(pt savepoint) {
	if pt.offset == p.pt.offset {
		return
	}
	p.pt = pt
}

// Cloner is implemented by any value that has a Clone method, which returns a
// copy of the value. This is mainly used for types which are not passed by
// value (e.g map, slice, chan) or structs that contain such types.
//
// This is used in conjunction with the global state feature to create proper
// copies of the state to allow the parser to properly restore the state in
// the case of backtracking.
type Cloner interface {
	Clone() interface{}
}

// clone and return parser current state.
func (p *parser) cloneState() storeDict {

	if len(p.cur.state) == 0 {
		if len(p.emptyState) > 0 {
			p.emptyState = make(storeDict)
		}
		return p.emptyState
	}

	state := make(storeDict, len(p.cur.state))
	for k, v := range p.cur.state {
		if c, ok := v.(Cloner); ok {
			state[k] = c.Clone()
		} else {
			state[k] = v
		}
	}
	return state
}

// restore parser current state to the state storeDict.
// every restoreState should applied only one time for every cloned state
func (p *parser) restoreState(state storeDict) {
	p.cur.state = state
}

// get the slice of bytes from the savepoint start to the current position.
func (p *parser) sliceFrom(start savepoint) []byte {
	return p.data[start.position.offset:p.pt.position.offset]
}

func (p *parser) buildRulesTable(g *grammar) {
	p.rules = make(map[string]*rule, len(g.rules))
	for _, r := range g.rules {
		p.rules[r.name] = r
	}
}

func (p *parser) parse(g *grammar) (val interface{}, err error) {
	if len(g.rules) == 0 {
		p.addErr(errNoRule)
		return nil, p.errs.err()
	}

	// TODO : not super critical but this could be generated
	p.buildRulesTable(g)

	if p.recover {
		// panic can be used in action code to stop parsing immediately
		// and return the panic as an error.
		defer func() {
			if e := recover(); e != nil {
				val = nil
				switch e := e.(type) {
				case error:
					p.addErr(e)
				default:
					p.addErr(fmt.Errorf("%v", e))
				}
				err = p.errs.err()
			}
		}()
	}

	startRule, ok := p.rules[p.entrypoint]
	if !ok {
		p.addErr(errInvalidEntrypoint)
		return nil, p.errs.err()
	}

	p.read() // advance to first rune
	val, ok = p.parseRule(startRule)
	if !ok {
		if len(*p.errs) == 0 {
			// If parsing fails, but no errors have been recorded, the expected values
			// for the farthest parser position are returned as error.
			maxFailExpectedMap := make(map[string]struct{}, len(p.maxFailExpected))
			for _, v := range p.maxFailExpected {
				maxFailExpectedMap[v] = struct{}{}
			}
			expected := make([]string, 0, len(maxFailExpectedMap))
			eof := false
			if _, ok := maxFailExpectedMap["!."]; ok {
				delete(maxFailExpectedMap, "!.")
				eof = true
			}
			for k := range maxFailExpectedMap {
				expected = append(expected, k)
			}
			sort.Strings(expected)
			if eof {
				expected = append(expected, "EOF")
			}
			p.addErrAt(errors.New("no match found, expected: "+listJoin(expected, ", ", "or")), p.maxFailPos, expected)
		}

		return nil, p.errs.err()
	}
	return val, p.errs.err()
}

func listJoin(list []string, sep string, lastSep string) string {
	switch len(list) {
	case 0:
		return ""
	case 1:
		return list[0]
	default:
		return fmt.Sprintf("%s %s %s", strings.Join(list[:len(list)-1], sep), lastSep, list[len(list)-1])
	}
}

func (p *parser) parseRule(rule *rule) (interface{}, bool) {
	p.rstack = append(p.rstack, rule)
	p.pushV()
	val, ok := p.parseExpr(rule.expr)
	p.popV()
	p.rstack = p.rstack[:len(p.rstack)-1]
	return val, ok
}

func (p *parser) parseExpr(expr interface{}) (interface{}, bool) {

	p.ExprCnt++
	if p.ExprCnt > p.maxExprCnt {
		panic(errMaxExprCnt)
	}

	var val interface{}
	var ok bool
	switch expr := expr.(type) {
	case *actionExpr:
		val, ok = p.parseActionExpr(expr)
	case *andCodeExpr:
		val, ok = p.parseAndCodeExpr(expr)
	case *andExpr:
		val, ok = p.parseAndExpr(expr)
	case *anyMatcher:
		val, ok = p.parseAnyMatcher(expr)
	case *charClassMatcher:
		val, ok = p.parseCharClassMatcher(expr)
	case *choiceExpr:
		val, ok = p.parseChoiceExpr(expr)
	case *labeledExpr:
		val, ok = p.parseLabeledExpr(expr)
	case *litMatcher:
		val, ok = p.parseLitMatcher(expr)
	case *notCodeExpr:
		val, ok = p.parseNotCodeExpr(expr)
	case *notExpr:
		val, ok = p.parseNotExpr(expr)
	case *oneOrMoreExpr:
		val, ok = p.parseOneOrMoreExpr(expr)
	case *recoveryExpr:
		val, ok = p.parseRecoveryExpr(expr)
	case *ruleRefExpr:
		val, ok = p.parseRuleRefExpr(expr)
	case *seqExpr:
		val, ok = p.parseSeqExpr(expr)
	case *stateCodeExpr:
		val, ok = p.parseStateCodeExpr(expr)
	case *throwExpr:
		val, ok = p.parseThrowExpr(expr)
	case *zeroOrMoreExpr:
		val, ok = p.parseZeroOrMoreExpr(expr)
	case *zeroOrOneExpr:
		val, ok = p.parseZeroOrOneExpr(expr)
	default:
		panic(fmt.Sprintf("unknown expression type %T", expr))
	}
	return val, ok
}

func (p *parser) parseActionExpr(act *actionExpr) (interface{}, bool) {
	start := p.pt
	val, ok := p.parseExpr(act.expr)
	if ok {
		p.cur.pos = start.position
		p.cur.text = p.sliceFrom(start)
		state := p.cloneState()
		actVal, err := act.run(p)
		if err != nil {
			p.addErrAt(err, start.position, []string{})
		}
		p.restoreState(state)

		val = actVal
	}
	return val, ok
}

func (p *parser) parseAndCodeExpr(and *andCodeExpr) (interface{}, bool) {
	state := p.cloneState()

	ok, err := and.run(p)
	if err != nil {
		p.addErr(err)
	}
	p.restoreState(state)

	return nil, ok
}

func (p *parser) parseAndExpr(and *andExpr) (interface{}, bool) {
	pt := p.pt
	state := p.cloneState()
	p.pushV()
	_, ok := p.parseExpr(and.expr)
	p.popV()
	p.restoreState(state)
	p.restore(pt)

	return nil, ok
}

func (p *parser) parseAnyMatcher(any *anyMatcher) (interface{}, bool) {
	if p.pt.rn == utf8.RuneError && p.pt.w == 0 {
		// EOF - see utf8.DecodeRune
		p.failAt(false, p.pt.position, ".")
		return nil, false
	}
	start := p.pt
	p.read()
	p.failAt(true, start.position, ".")
	return p.sliceFrom(start), true
}

func (p *parser) parseCharClassMatcher(chr *charClassMatcher) (interface{}, bool) {
	cur := p.pt.rn
	start := p.pt

	if cur < 128 {
		if chr.basicLatinChars[cur] != chr.inverted {
			p.read()
			p.failAt(true, start.position, chr.val)
			return p.sliceFrom(start), true
		}
		p.failAt(false, start.position, chr.val)
		return nil, false
	}

	// can't match EOF
	if cur == utf8.RuneError && p.pt.w == 0 { // see utf8.DecodeRune
		p.failAt(false, start.position, chr.val)
		return nil, false
	}

	if chr.ignoreCase {
		cur = unicode.ToLower(cur)
	}

	// try to match in the list of available chars
	for _, rn := range chr.chars {
		if rn == cur {
			if chr.inverted {
				p.failAt(false, start.position, chr.val)
				return nil, false
			}
			p.read()
			p.failAt(true, start.position, chr.val)
			return p.sliceFrom(start), true
		}
	}

	// try to match in the list of ranges
	for i := 0; i < len(chr.ranges); i += 2 {
		if cur >= chr.ranges[i] && cur <= chr.ranges[i+1] {
			if chr.inverted {
				p.failAt(false, start.position, chr.val)
				return nil, false
			}
			p.read()
			p.failAt(true, start.position, chr.val)
			return p.sliceFrom(start), true
		}
	}

	// try to match in the list of Unicode classes
	for _, cl := range chr.classes {
		if unicode.Is(cl, cur) {
			if chr.inverted {
				p.failAt(false, start.position, chr.val)
				return nil, false
			}
			p.read()
			p.failAt(true, start.position, chr.val)
			return p.sliceFrom(start), true
		}
	}

	if chr.inverted {
		p.read()
		p.failAt(true, start.position, chr.val)
		return p.sliceFrom(start), true
	}
	p.failAt(false, start.position, chr.val)
	return nil, false
}

func (p *parser) parseChoiceExpr(ch *choiceExpr) (interface{}, bool) {
	for altI, alt := range ch.alternatives {
		// dummy assignment to prevent compile error if optimized
		_ = altI

		state := p.cloneState()

		p.pushV()
		val, ok := p.parseExpr(alt)
		p.popV()
		if ok {
			return val, ok
		}
		p.restoreState(state)
	}
	return nil, false
}

func (p *parser) parseLabeledExpr(lab *labeledExpr) (interface{}, bool) {
	p.pushV()
	val, ok := p.parseExpr(lab.expr)
	p.popV()
	if ok && lab.label != "" {
		m := p.vstack[len(p.vstack)-1]
		m[lab.label] = val
	}
	return val, ok
}

func (p *parser) parseLitMatcher(lit *litMatcher) (interface{}, bool) {
	ignoreCase := ""
	if lit.ignoreCase {
		ignoreCase = "i"
	}
	val := fmt.Sprintf("%q%s", lit.val, ignoreCase)
	start := p.pt
	for _, want := range lit.val {
		cur := p.pt.rn
		if lit.ignoreCase {
			cur = unicode.ToLower(cur)
		}
		if cur != want {
			p.failAt(false, start.position, val)
			p.restore(start)
			return nil, false
		}
		p.read()
	}
	p.failAt(true, start.position, val)
	return p.sliceFrom(start), true
}

func (p *parser) parseNotCodeExpr(not *notCodeExpr) (interface{}, bool) {
	state := p.cloneState()

	ok, err := not.run(p)
	if err != nil {
		p.addErr(err)
	}
	p.restoreState(state)

	return nil, !ok
}

func (p *parser) parseNotExpr(not *notExpr) (interface{}, bool) {
	pt := p.pt
	state := p.cloneState()
	p.pushV()
	p.maxFailInvertExpected = !p.maxFailInvertExpected
	_, ok := p.parseExpr(not.expr)
	p.maxFailInvertExpected = !p.maxFailInvertExpected
	p.popV()
	p.restoreState(state)
	p.restore(pt)

	return nil, !ok
}

func (p *parser) parseOneOrMoreExpr(expr *oneOrMoreExpr) (interface{}, bool) {
	var vals []interface{}

	for {
		p.pushV()
		val, ok := p.parseExpr(expr.expr)
		p.popV()
		if !ok {
			if len(vals) == 0 {
				// did not match once, no match
				return nil, false
			}
			return vals, true
		}
		vals = append(vals, val)
	}
}

func (p *parser) parseRecoveryExpr(recover *recoveryExpr) (interface{}, bool) {

	p.pushRecovery(recover.failureLabel, recover.recoverExpr)
	val, ok := p.parseExpr(recover.expr)
	p.popRecovery()

	return val, ok
}

func (p *parser) parseRuleRefExpr(ref *ruleRefExpr) (interface{}, bool) {
	if ref.name == "" {
		panic(fmt.Sprintf("%s: invalid rule: missing name", ref.pos))
	}

	rule := p.rules[ref.name]
	if rule == nil {
		p.addErr(fmt.Errorf("undefined rule: %s", ref.name))
		return nil, false
	}
	return p.parseRule(rule)
}

func (p *parser) parseSeqExpr(seq *seqExpr) (interface{}, bool) {
	vals := make([]interface{}, 0, len(seq.exprs))

	pt := p.pt
	state := p.cloneState()
	for _, expr := range seq.exprs {
		val, ok := p.parseExpr(expr)
		if !ok {
			p.restoreState(state)
			p.restore(pt)
			return nil, false
		}
		vals = append(vals, val)
	}
	return vals, true
}

func (p *parser) parseStateCodeExpr(state *stateCodeExpr) (interface{}, bool) {
	err := state.run(p)
	if err != nil {
		p.addErr(err)
	}
	return nil, true
}

func (p *parser) parseThrowExpr(expr *throwExpr) (interface{}, bool) {

	for i := len(p.recoveryStack) - 1; i >= 0; i-- {
		if recoverExpr, ok := p.recoveryStack[i][expr.label]; ok {
			if val, ok := p.parseExpr(recoverExpr); ok {
				return val, ok
			}
		}
	}

	return nil, false
}

func (p *parser) parseZeroOrMoreExpr(expr *zeroOrMoreExpr) (interface{}, bool) {
	var vals []interface{}

	for {
		p.pushV()
		val, ok := p.parseExpr(expr.expr)
		p.popV()
		if !ok {
			return vals, true
		}
		vals = append(vals, val)
	}
}

func (p *parser) parseZeroOrOneExpr(expr *zeroOrOneExpr) (interface{}, bool) {
	p.pushV()
	val, _ := p.parseExpr(expr.expr)
	p.popV()
	// whether it matched or not, consider it a match
	return val, true
}

func rangeTable(class string) *unicode.RangeTable {
	if rt, ok := unicode.Categories[class]; ok {
		return rt
	}
	if rt, ok := unicode.Properties[class]; ok {
		return rt
	}
	if rt, ok := unicode.Scripts[class]; ok {
		return rt
	}

	// cannot happen
	panic(fmt.Sprintf("invalid Unicode class: %s", class))
}
