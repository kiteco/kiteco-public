package source

// EMRDatasetOpts for configuring EMRDataset
type EMRDatasetOpts struct {
	DatasetOpts
	MaxRecords  int
	MaxFileSize int
}

// DefaultEMRDatasetOpts ...
var DefaultEMRDatasetOpts = EMRDatasetOpts{
	DatasetOpts: DefaultDatasetOpts,
	MaxFileSize: 500000,
}

// NewEMRDataset that reads from each provided file, the `Sample`s that are output
// are of type `pipeline.Keyed`
// DEPRECATED: just call NewDataset directly
func NewEMRDataset(name string, opts EMRDatasetOpts, files []string) *Dataset {
	pf := EMRProcessFn(opts.MaxRecords, opts.MaxFileSize)
	return NewDataset(opts.DatasetOpts, name, pf, files...)
}
