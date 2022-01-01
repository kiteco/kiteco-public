package editorapi

// AutocorrectMetaData data sent with each request.
type AutocorrectMetaData struct {
	Event         string `json:"event"`
	Source        string `json:"source"`
	OS            string `json:"os_name"`
	PluginVersion string `json:"plugin_version"`
}

// AutocorrectRequest sent to the autocorrect endpoint.
type AutocorrectRequest struct {
	MetaData AutocorrectMetaData `json:"metadata"`
	Filename string              `json:"filename"`
	Buffer   string              `json:"buffer"`
	Language string              `json:"language"`
}

// AutocorrectResponse returned from the autocorrect endpoint.
type AutocorrectResponse struct {
	Filename            string            `json:"filename"`
	RequestedBufferHash string            `json:"requested_buffer_hash"`
	NewBuffer           string            `json:"new_buffer"`
	Diffs               []AutocorrectDiff `json:"diffs"`
	Version             uint64            `json:"version"`
}

// AutocorrectDiff for a file.
type AutocorrectDiff struct {
	Inserted             []AutocorrectDiffLine `json:"inserted"`
	Deleted              []AutocorrectDiffLine `json:"deleted"`
	NewBufferOffsetRunes uint64                `json:"new_buffer_offset_runes"`
	NewBufferOffsetBytes uint64                `json:"new_buffer_offset_bytes"`
}

// AutocorrectDiffLine in a file.
type AutocorrectDiffLine struct {
	Text     string                    `json:"text"`
	Line     int                       `json:"line"` // 0 based
	Emphasis []AutocorrectLineEmphasis `json:"emphasis"`
}

// AutocorrectLineEmphasis indicates which portions of a line should
// be emphasized on the frontend
type AutocorrectLineEmphasis struct {
	StartBytes uint64 `json:"start_bytes"`
	EndBytes   uint64 `json:"end_bytes"`
	StartRunes uint64 `json:"start_runes"`
	EndRunes   uint64 `json:"end_runes"`
}

// AutocorrectModelInfoRequest sent to the model info endpoint.
type AutocorrectModelInfoRequest struct {
	MetaData AutocorrectMetaData `json:"metadata"`
	Language string              `json:"language"`
	Version  uint64              `json:"version"`
}

// AutocorrectModelInfoResponse contains information about the model
// used for autocorrect
type AutocorrectModelInfoResponse struct {
	DateShipped int                  `json:"date_shipped"`
	Examples    []AutocorrectExample `json:"examples"`
}

// AutocorrectExample contains information about a specific
// example in which autocorrect will make a correction.
type AutocorrectExample struct {
	Synopsis string                   `json:"synopsis"`
	Old      []AutocorrectExampleLine `json:"old"`
	New      []AutocorrectExampleLine `json:"new"`
}

// AutocorrectExampleLine is a line in an autocorrect example
type AutocorrectExampleLine struct {
	Text     string                    `json:"text"`
	Emphasis []AutocorrectLineEmphasis `json:"emphasis"`
}
