package pythonbatch

import "github.com/kiteco/kiteco/kite-golib/status"

var (
	indexSection = status.NewSection("lang/python/pythonbatch (SymbolIndex stats)")

	filesPerIndex             = indexSection.SampleInt64("Input Files")
	valuesPerIndex            = indexSection.SampleInt64("Values")
	modulesPerIndex           = indexSection.SampleInt64("Modules")
	pkgsPerIndex              = indexSection.SampleInt64("Packages")
	defsPerIndex              = indexSection.SampleInt64("Definitions")
	docsPerIndex              = indexSection.SampleInt64("Documentation")
	argSpecsPerIndex          = indexSection.SampleInt64("Argspecs")
	methodPatternsPerIndex    = indexSection.SampleInt64("Method Patterns")
	signaturePatternsPerIndex = indexSection.SampleInt64("Signature Patterns")
	invertedTokensPerIndex    = indexSection.SampleInt64("Active Search Index Tokens")
	invertedBytesPerIndex     = indexSection.SampleByte("Active Search Index Size")

	filesPerIndexLibs   = indexSection.SampleInt64("Library files")
	valuesPerIndexLibs  = indexSection.SampleInt64("Library values")
	modulesPerIndexLibs = indexSection.SampleInt64("Library modules")
	pkgsPerIndexLibs    = indexSection.SampleInt64("Library packages")
)
var (
	buildSection = status.NewSection("lang/python/pythonbatch (Builder)")

	selectFilesDuration     = buildSection.SampleDuration("SelectFiles")
	selectLibrariesDuration = buildSection.SampleDuration("SelectLibraties")
	prefetchFilesDuration   = buildSection.SampleDuration("Prefetch files")
	managerAddDuration      = buildSection.SampleDuration("Adding files to Manager")
	managerBuildDuration    = buildSection.SampleDuration("Manager Build")
	flattenDuration         = buildSection.SampleDuration("Flatten")
	putDuration             = buildSection.SampleDuration("Put artifacts")
	tooLargeRatio           = buildSection.Ratio("File dropped (too large)")
)
