package pythontracking

import "github.com/kiteco/kiteco/kite-go/lang/python/pythonlocal"

// EventType is used to distinguish between different types of tracking events, all of which use pythontracking.Event
// as the top-level event.
type EventType string

const (
	// ServerSignatureFailureEvent is logged by the usernode when a failure occurs in returning a signature
	ServerSignatureFailureEvent EventType = "server_signature_failure"
	// ServerCompletionsFailureEvent is logged by the usernode when a failure occurs in providing completions
	ServerCompletionsFailureEvent EventType = "server_completions_failure"
)

// CompletionsFailure represents a type of failure in providing completions
type CompletionsFailure string

const (
	// NoCompletionsFailure represents a failure in which completions were expected but not provided
	NoCompletionsFailure CompletionsFailure = "no_completions"
	// CompletionsSample represents a sampling of successful completions (not actually a failure)
	CompletionsSample CompletionsFailure = "sample"
)

// Completions describes details about a completions-related event
type Completions struct {
	Failure CompletionsFailure `json:"failure"`
}

// CalleeFailure represents a type of failure in the callee endpoint
type CalleeFailure string

const (
	// NoContextFailure is a CalleeFailure
	NoContextFailure CalleeFailure = "no_context"
	// OffsetParseFailure is a CalleeFailure
	OffsetParseFailure CalleeFailure = "error_parsing_offset"
	// OutsideParensFailure is a CalleeFailure
	OutsideParensFailure CalleeFailure = "call_expr_outside_parens"
	// NoCallExprFailure is a CalleeFailure
	NoCallExprFailure CalleeFailure = "call_expr_not_found"
	// NilRefFailure is a CalleeFailure
	NilRefFailure CalleeFailure = "nil_reference"
	// UnresolvedValueFailure is a CalleeFailure
	UnresolvedValueFailure CalleeFailure = "unresolved_value"
	// InvalidKindFailure is a CalleeFailure
	InvalidKindFailure CalleeFailure = "invalid_callee_kind"
	// NoSignaturesFailure is a CalleeFailure
	NoSignaturesFailure CalleeFailure = "no_signatures"
	// JSONMarshalFailure is a CalleeFailure
	JSONMarshalFailure CalleeFailure = "error_mashalling_json"
	// ValTranslateFailure is a CalleeFailure
	ValTranslateFailure CalleeFailure = "nil_translated_val"
)

// Callee describes details about a signature experience.
type Callee struct {
	Failure   CalleeFailure       `json:"failure"`
	ReqParams map[string][]string `json:"req_params"`
}

// Event describes an event that occurs, along with enough information to rebuild the relevant python.Context.
type Event struct {
	Type      EventType `json:"type"`
	MetricsID string    `json:"metrics_id"`
	UserID    int64     `json:"user"`
	MachineID string    `json:"machine"`
	Filename  string    `json:"filename"`
	Buffer    string    `json:"buffer"`
	Offset    int64     `json:"offset"`

	ArtifactMeta struct {
		Error               string            `json:"error"`
		OriginatingFilename string            `json:"originating_filename"`
		FileHashes          map[string]string `json:"file_hashes"`
		MissingHashes       map[string]bool   `json:"missing_hashes"`
	} `json:"artifact_metadata"`

	// Region will be set to the REGION env variable when TrackFailure is called
	Region string `json:"region"`
	// LocalFilesBucket will be set to the LOCALFILES_S3_BUCKET env variable when TrackFailure is called
	LocalFilesBucket string `json:"local_files_bucket"`

	// Callee contains callee-specific data, if relevant
	Callee *Callee `json:"callee,omitempty"`

	// Completions contains completions-specific data, if relevant
	Completions *Completions `json:"completions,omitempty"`
}

// Failure attempts to find a failure reason associated with the event.
func (event *Event) Failure() string {
	if event.Callee != nil {
		return string(event.Callee.Failure)
	}
	if event.Completions != nil {
		return string(event.Completions.Failure)
	}
	return ""
}

// SetIndex sets the relevant fields of the event given a local index.
func (event *Event) SetIndex(index *pythonlocal.SymbolIndex, artifactError error) {
	if index != nil {
		event.ArtifactMeta.OriginatingFilename = index.ArtifactMetadata.OriginatingFilename
		event.ArtifactMeta.FileHashes = index.ArtifactMetadata.FileHashes
		event.ArtifactMeta.MissingHashes = index.ArtifactMetadata.MissingHashes
	}
	if artifactError != nil {
		event.ArtifactMeta.Error = artifactError.Error()
	}
}
