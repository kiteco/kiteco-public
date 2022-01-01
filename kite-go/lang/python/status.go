package python

import "github.com/kiteco/kiteco/kite-golib/status"

var (
	section = status.NewSection("lang/python")

	relexRatio        = section.Ratio("Relex")
	parseErrorRatio   = section.Ratio("Parse error")
	reuseContextRatio = section.Ratio("Reuse context")

	haveLocalGraph = section.Ratio("Have local graph")

	editMatchNodeRatio = section.Ratio("Cursor was on a NameExpr, AttributeExpr, CallExpr or DottedExpr (edit event)")
	editResolveRatio   = section.Ratio("Node was resolved (edit event)")
	editNodeType       = section.Breakdown("AST node type (edit event)")

	selectionMatchNodeRatio = section.Ratio("Cursor was on a NameExpr, AttributeExpr, CallExpr or DottedExpr (selection event)")
	selectionResolveRatio   = section.Ratio("Node was resolved (selection event)")
	selectionNodeType       = section.Breakdown("AST node type (selection event)")

	otherMatchNodeRatio = section.Ratio("Cursor was on a NameExpr, AttributeExpr, CallExpr or DottedExpr (other event)")
	otherResolveRatio   = section.Ratio("Node was resolved (other event)")
	otherNodeType       = section.Breakdown("AST node type (other event)")

	handleEventDuration = section.SampleDuration("HandleEvent()")
	bufferDuration      = section.SampleDuration("HandleEvent().file.HandleEvent()")
	contextDuration     = section.SampleDuration("HandleEvent().parse()")
	handleDuration      = section.SampleDuration("HandleEvent().handle()")
	newContextDuration  = section.SampleDuration("HandleEvent().parse().NewContext()")

	parseDuration       = section.SampleDuration("NewContext().Parse()")
	resolveDuration     = section.SampleDuration("NewContext().Resolve()")
	bufferIndexDuration = section.SampleDuration("NewContext().newBufferIndex()")
	resolveTimeout      = section.Ratio("NewContext().Resolve() timeout")

	bufferSizeSample = section.SampleByte("Python buffer size")
)

var (
	endpointsSection = status.NewSection("Python API Quality: Endpoints (All)")

	valueCounter          = endpointsSection.Counter("Value endpoint")
	membersCounter        = endpointsSection.Counter("Members endpoint")
	symbolCounter         = endpointsSection.Counter("Symbol endpoint")
	hoverCounter          = endpointsSection.Counter("Hover endpoint")
	hoverSelectionCounter = endpointsSection.Counter("Hover endpoint (selection query)")
	calleeCounter         = endpointsSection.Counter("Callee endpoint")

	valueDuration   = endpointsSection.SampleDuration("Value endpoint")
	membersDuration = endpointsSection.SampleDuration("Members endpoint")
	symbolDuration  = endpointsSection.SampleDuration("Symbol endpoint")
	hoverDuration   = endpointsSection.SampleDuration("Hover endpoint")
	calleeDuration  = endpointsSection.SampleDuration("Callee endpoint")

	// BufferStatusCode records the status code returned by all buffer endpoints
	BufferStatusCode = endpointsSection.Breakdown("Status codes: All buffer endpoints")

	valueStatusCode   = endpointsSection.Breakdown("Status codes: Value endpoint")
	membersStatusCode = endpointsSection.Breakdown("Status codes: Members endpoint")
	symbolStatusCode  = endpointsSection.Breakdown("Status codes: Symbol endpoint")
	hoverStatusCode   = endpointsSection.Breakdown("Status codes: Hover endpoint")
	calleeStatusCode  = endpointsSection.Breakdown("Status codes: Callee endpoint")

	missingCalleeReason = endpointsSection.Breakdown("Missing callee reasons")

	// CalleeStatePresent records the ratio of non-410's to the callee endpoint
	CalleeStatePresent = endpointsSection.Ratio("Callee state not gone")

	// BufferStateHistory records ratio of states that arrived before the state changed before the request was made
	BufferStateHistory = endpointsSection.Ratio("Buffer state found in history")

	// BufferStateHistoryDuration records how far off the request timing was
	BufferStateHistoryDuration = endpointsSection.SampleDuration("Buffer state found in history")

	// BufferStateAfter records the ratio of states that arrived too late, e.g after the request
	BufferStateAfter = endpointsSection.Ratio("Buffer state matched after request")

	// BufferStateAfterDuration records how far off the request timing was
	BufferStateAfterDuration = endpointsSection.SampleDuration("Buffer state matched after request")
)

var (
	hoverEndpointCoverage = status.NewSection("Python API Quality: Endpoints (Hover)")

	hoverIndexAvailableRatio = hoverEndpointCoverage.Ratio("Index Available")

	hoverFailReason = hoverEndpointCoverage.Breakdown("Fail Reason")
)

var (
	valueCoverageSection = status.NewSection("Python API Quality: Coverage (Value ID)")

	validValueRatio = valueCoverageSection.Ratio("Valid Value ID")

	invalidValueSource = valueCoverageSection.Breakdown("Invalid Value ID: Sources")
	invalidValueKind   = valueCoverageSection.Breakdown("Invalid Value ID: Kinds")
)

var (
	symbolCoverageSection = status.NewSection("Python API Quality: Coverage (Symbol ID)")
	validSymbolRatio      = symbolCoverageSection.Ratio("Valid Symbol ID")

	invalidSymbolSource                    = symbolCoverageSection.Breakdown("Invalid Symbol ID: Sources")
	invalidLocalSymbolValueKind            = symbolCoverageSection.Breakdown("Invalid Symbol ID (source = local): Value kinds")
	invalidMissingNamespaceSymbolValueKind = symbolCoverageSection.Breakdown("Invalid Symbol ID (source = missing namespace): Value kinds")
	invalidUndefinedSymbolValueKind        = symbolCoverageSection.Breakdown("Invalid Symbol ID (source = undefined): Value kinds")
)

var (
	bufferedSymbolCoverageSection = status.NewSection("Python API Quality: Coverage (Symbol ID w/ Buffer Index)")

	invalidBufferedSymbolSource                    = bufferedSymbolCoverageSection.Breakdown("Invalid Symbol ID: Sources")
	invalidMissingNamespaceBufferedSymbolValueKind = bufferedSymbolCoverageSection.Breakdown("Invalid Symbol ID (source = missing namespace): Value kinds")
)

var (
	unbufferedSymbolCoverageSection = status.NewSection("Python API Quality: Coverage (Symbol ID w/o Buffer Index)")

	invalidUnbufferedSymbolSource                    = unbufferedSymbolCoverageSection.Breakdown("Invalid Symbol ID: Sources")
	invalidMissingNamespaceUnbufferedSymbolValueKind = unbufferedSymbolCoverageSection.Breakdown("Invalid Symbol ID (source = missing namespace): Value kinds")
)

func init() {
	editNodeType.AddCategories("NameExpr", "AttributeExpr", "CallExpr")
	selectionNodeType.AddCategories("NameExpr", "AttributeExpr", "CallExpr")
	otherNodeType.AddCategories("NameExpr", "AttributeExpr", "CallExpr")
}
