package response

// Timeline is a response for a timeline of user events (i.e., a fragment of
// their journal).
type Timeline struct {
	Type     string        `json:"type"`
	Segments []interface{} `json:"segments"`
}

// CodeSegment is a response for a diff of code as a single event in a Timeline.
type CodeSegment struct {
	Type      string `json:"type"`
	Timestamp int64  `json:"timestamp"`
	Filename  string `json:"filename"`
	Code1     string `json:"code1"`
	Code2     string `json:"code2"`
}

// TerminalSegment is a response for a terminal's command/output pair, for a
// single event in a Timeline.
type TerminalSegment struct {
	Type      string `json:"type"`
	Timestamp int64  `json:"timestamp"`
	Command   string `json:"command"`
	Output    string `json:"output"`
}
