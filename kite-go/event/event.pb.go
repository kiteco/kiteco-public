/*
It has these top-level messages:
	Selection
	Diff
	Event
*/
package event

import (
	"fmt"
)

type DiffType int32

const (
	DiffType_NONE   DiffType = 0
	DiffType_INSERT DiffType = 1
	DiffType_DELETE DiffType = 2
)

var DiffType_name = map[int32]string{
	0: "NONE",
	1: "INSERT",
	2: "DELETE",
}
var DiffType_value = map[string]int32{
	"NONE":   0,
	"INSERT": 1,
	"DELETE": 2,
}

func (x DiffType) Enum() *DiffType {
	p := new(DiffType)
	*p = x
	return p
}
func (x DiffType) String() string {
	return DiffType_name[int32(x)]
}

type Selection struct {
	Start            *int64 `json:"start,omitempty"`
	End              *int64 `json:"end,omitempty"`
	XXX_unrecognized []byte `json:"-"`
}

func (m *Selection) Reset()         { *m = Selection{} }
func (m *Selection) String() string { return fmt.Sprintf("selection %d-%d", m.GetStart(), m.GetEnd()) }
func (*Selection) ProtoMessage()    {}

func (m *Selection) GetStart() int64 {
	if m != nil && m.Start != nil {
		return *m.Start
	}
	return 0
}

func (m *Selection) GetEnd() int64 {
	if m != nil && m.End != nil {
		return *m.End
	}
	return 0
}

type Diff struct {
	Type             *DiffType `json:"type,omitempty"`
	Offset           *int32    `json:"offset,omitempty"`
	Text             *string   `json:"text,omitempty"`
	XXX_unrecognized []byte    `json:"-"`
}

func (m *Diff) Reset()         { *m = Diff{} }
func (m *Diff) String() string { return m.GetText() }
func (*Diff) ProtoMessage()    {}

const Default_Diff_Type DiffType = DiffType_NONE

func (m *Diff) GetType() DiffType {
	if m != nil && m.Type != nil {
		return *m.Type
	}
	return Default_Diff_Type
}

func (m *Diff) GetOffset() int32 {
	if m != nil && m.Offset != nil {
		return *m.Offset
	}
	return 0
}

func (m *Diff) GetText() string {
	if m != nil && m.Text != nil {
		return *m.Text
	}
	return ""
}

type Event struct {
	UserId             *int64       `json:"userId,omitempty"`
	Source             *string      `json:"source,omitempty"`
	Action             *string      `json:"action,omitempty"`
	Filename           *string      `json:"filename,omitempty"`
	Text               *string      `json:"text,omitempty"`
	Selections         []*Selection `json:"selections,omitempty"`
	Diffs              []*Diff      `json:"diffs,omitempty"`
	Command            *string      `json:"command,omitempty"`
	Output             *string      `json:"output,omitempty"`
	Timestamp          *int64       `json:"timestamp,omitempty"`
	PluginId           *string      `json:"pluginId,omitempty"`
	Id                 *int64       `json:"id,omitempty"`
	MachineId          *string      `json:"machineId,omitempty"`
	TextChecksum       *uint64      `json:"textChecksum,omitempty"`
	ProcessingTime     *int64       `json:"processingTime,omitempty"`
	PingLatency        *int64       `json:"pingLatency,omitempty"`
	LastEventLatency   *int64       `json:"lastEventLatency,omitempty"`
	LastBackendLatency *int64       `json:"lastBackendLatency,omitempty"`
	ClientVersion      *string      `json:"clientVersion,omitempty"`
	LastResponseSize   *int64       `json:"lastResponseSize,omitempty"`
	Ip                 *string      `json:"ip,omitempty"`
	CodeColumns        *int64       `json:"codeColumns,omitempty"`
	TextMD5            *string      `json:"textMD5,omitempty"`
	InitialAction      *string      `json:"initialAction,omitempty"`
	FirstSeen          *int64       `json:"firstSeen,omitempty"`
	ReferenceState     *string      `json:"referenceState,omitempty"`
}

func (m *Event) Reset()         { *m = Event{} }
func (m *Event) String() string { return fmt.Sprintf("Event %d: %s", m.GetId(), m.GetText()) }
func (*Event) ProtoMessage()    {}

func (m *Event) GetUserId() int64 {
	if m != nil && m.UserId != nil {
		return *m.UserId
	}
	return 0
}

func (m *Event) GetSource() string {
	if m != nil && m.Source != nil {
		return *m.Source
	}
	return ""
}

func (m *Event) GetAction() string {
	if m != nil && m.Action != nil {
		return *m.Action
	}
	return ""
}

func (m *Event) GetFilename() string {
	if m != nil && m.Filename != nil {
		return *m.Filename
	}
	return ""
}

func (m *Event) GetText() string {
	if m != nil && m.Text != nil {
		return *m.Text
	}
	return ""
}

func (m *Event) GetSelections() []*Selection {
	if m != nil {
		return m.Selections
	}
	return nil
}

func (m *Event) GetDiffs() []*Diff {
	if m != nil {
		return m.Diffs
	}
	return nil
}

func (m *Event) GetCommand() string {
	if m != nil && m.Command != nil {
		return *m.Command
	}
	return ""
}

func (m *Event) GetOutput() string {
	if m != nil && m.Output != nil {
		return *m.Output
	}
	return ""
}

func (m *Event) GetTimestamp() int64 {
	if m != nil && m.Timestamp != nil {
		return *m.Timestamp
	}
	return 0
}

func (m *Event) GetPluginId() string {
	if m != nil && m.PluginId != nil {
		return *m.PluginId
	}
	return ""
}

func (m *Event) GetId() int64 {
	if m != nil && m.Id != nil {
		return *m.Id
	}
	return 0
}

func (m *Event) GetMachineId() string {
	if m != nil && m.MachineId != nil {
		return *m.MachineId
	}
	return ""
}

func (m *Event) GetTextChecksum() uint64 {
	if m != nil && m.TextChecksum != nil {
		return *m.TextChecksum
	}
	return 0
}

func (m *Event) GetProcessingTime() int64 {
	if m != nil && m.ProcessingTime != nil {
		return *m.ProcessingTime
	}
	return 0
}

func (m *Event) GetPingLatency() int64 {
	if m != nil && m.PingLatency != nil {
		return *m.PingLatency
	}
	return 0
}

func (m *Event) GetLastEventLatency() int64 {
	if m != nil && m.LastEventLatency != nil {
		return *m.LastEventLatency
	}
	return 0
}

func (m *Event) GetLastBackendLatency() int64 {
	if m != nil && m.LastBackendLatency != nil {
		return *m.LastBackendLatency
	}
	return 0
}

func (m *Event) GetClientVersion() string {
	if m != nil && m.ClientVersion != nil {
		return *m.ClientVersion
	}
	return ""
}

func (m *Event) GetLastResponseSize() int64 {
	if m != nil && m.LastResponseSize != nil {
		return *m.LastResponseSize
	}
	return 0
}

func (m *Event) GetIp() string {
	if m != nil && m.Ip != nil {
		return *m.Ip
	}
	return ""
}

func (m *Event) GetCodeColumns() int64 {
	if m != nil && m.CodeColumns != nil {
		return *m.CodeColumns
	}
	return 0
}

func (m *Event) GetTextMD5() string {
	if m != nil && m.TextMD5 != nil {
		return *m.TextMD5
	}
	return ""
}

func (m *Event) GetInitialAction() string {
	if m != nil && m.InitialAction != nil {
		return *m.InitialAction
	}
	return ""
}

func (m *Event) GetFirstSeen() int64 {
	if m != nil && m.FirstSeen != nil {
		return *m.FirstSeen
	}
	return 0
}

func (m *Event) GetReferenceState() string {
	if m != nil && m.ReferenceState != nil {
		return *m.ReferenceState
	}
	return ""
}
