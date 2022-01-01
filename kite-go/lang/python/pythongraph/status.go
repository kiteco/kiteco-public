package pythongraph

import "github.com/kiteco/kiteco/kite-golib/status"

// StatusSectionName is the name of the status section
// for this package.
var StatusSectionName = "/lang/python/pythongraph"

var (
	section = status.NewSection(StatusSectionName)

	newBuilderDuration = section.SampleDuration("New Builder")

	buildEdgesDuration = section.SampleDuration("Build Edges")

	predictExprDuration = section.SampleDuration("Predict Expr")

	predictExprAttrBaseDuration = section.SampleDuration("Predict Expr Attr Base")

	predictExprAttrDuration = section.SampleDuration("Predict Expr Attr")

	predictExprCallDuration = section.SampleDuration("Predict Expr Call")

	predictExprInferNameDuration = section.SampleDuration("Predict Expr Infer Name")

	buildContextGraphDuration = section.SampleDuration("Build context graph")
)
