package pythoncode

const (
	// DefaultPipelineRoot is the root of the python-code-example EMR pipeline
	DefaultPipelineRoot = "s3://kite-emr/users/tarak/python-code-examples/2016-01-21_15-47-59-PM/"
	// DefaultUnfilteredPackageStats is the path to the default github package stats, some of which may not appear in the import graph
	DefaultUnfilteredPackageStats = "s3://kite-emr/users/juan/python-module-stats/2016-07-20_20-40-54-PM/module-stats/output"
	// DefaultPackageStats is the path to the default github package stats that correspond to nodes in the import graph
	// TODO(damian): This dataset should be eventually deprecated. The SymbolCounts resource contains the same stats,
	// but for a newer run and with a finer-grained breakdown
	DefaultPackageStats = "s3://kite-emr/users/juan/python-module-stats/2016-07-20_20-40-54-PM/in-graph/output"
	// DefaultSignaturePatterns is the path to the default signature patterns.
	DefaultSignaturePatterns = "s3://kite-emr/users/juan/python-signature-patterns/2016-07-29_14-45-39-PM/merge-signature-patterns/output/part-00000"
	// DefaultPackageCooccurences is the path to the default package cooccurence data
	DefaultPackageCooccurences = "s3://kite-emr/users/juan/python-module-cooccurence/2016-07-12_21-37-54-PM/merge/output/part-00000"
	// DefaultKwargs is the path to the default possible **kwargs.
	DefaultKwargs = "s3://kite-emr/users/juan/python-signature-patterns/2016-07-29_14-45-39-PM/extract-kwargs/output/part-00000"

	// old DedupedCodeDumpPath = "s3://kite-emr/users/juan/python-dedupe-code/2018-07-26_13-53-43-PM/dedupe/output/"

	// DedupedCodeDumpPath is the path to the new dump of deduped python source code from github.
	DedupedCodeDumpPath = "s3://kite-local-pipelines/gh-dump-python/2019-04-30_03-46-49-AM"

	// HashToSourceIndexPath is the path to the hash to source index for the new gh dump.
	HashToSourceIndexPath = "s3://kite-local-pipelines/python-hash-to-source-index/2019-07-02_02-26-59-AM"

	symbolToHashesRoot = "s3://kite-local-pipelines/python-symbol-to-hashes-index/2020-07-08_11-33-44-AM"
	// SymbolToHashesIndexPath is the path to the symbol to hashes index for the new gh dump
	SymbolToHashesIndexPath = symbolToHashesRoot + "/symbols"
	// CanonicalSymbolToHashesIndexPath is the path to the canonical symbol to hashes index for the new gh dump
	CanonicalSymbolToHashesIndexPath = symbolToHashesRoot + "/canonical-symbols"

	// KeywordCountsStats is the stats of keyword arguments frequency for function calls
	KeywordCountsStats = "s3://kite-local-pipelines/python-extract-keyword-parameters/2019-05-10_04-44-19-PM/stats.gob"

	// CallPatterns are the raw call patterns extracted from github
	CallPatterns = "s3://kite-local-pipelines/python-call-patterns/2019-07-29_04-46-16-PM"

	// TypeInductionTrainData is the training samples for type induction
	TypeInductionTrainData = "s3://kite-local-pipelines/type-induction/traindata/2019-08-01_18-04-20PM"

	// TypeInductionValidateData is the validation examples extracted from segment data
	TypeInductionValidateData = "s3://kite-local-pipelines/type-induction/validatedata/2019-07-30_23-15-32PM"

	// EMModelRoot is the root of em models for return types
	EMModelRoot = "s3://kite-local-pipelines/type-induction/model/2019-08-26_06-33-47-AM"
)
