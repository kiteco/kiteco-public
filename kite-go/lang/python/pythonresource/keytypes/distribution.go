package keytypes

// BuiltinDistributionName is the name for the builtin pseudo-distribution
const BuiltinDistributionName = "builtin-stdlib"

var (
	// BuiltinDistribution3 is the distribution for Python 3 builtins
	BuiltinDistribution3 = Distribution{
		Name:    BuiltinDistributionName,
		Version: "3.7",
	}

	// AlembicDistribution ...
	AlembicDistribution = Distribution{
		Name:    "alembic",
		Version: "1.0.10",
	}

	// NumpyDistribution ...
	NumpyDistribution = Distribution{
		Name:    "numpy",
		Version: "1.16.4",
	}

	// RequestsDistribution ...
	RequestsDistribution = Distribution{
		Name:    "requests",
		Version: "2.22.0",
	}

	// TensorflowDistribution ...
	TensorflowDistribution = Distribution{
		Name:    "tensorflow",
		Version: "1.13.1",
	}

	// BotoDistribution ...
	BotoDistribution = Distribution{
		Name:    "boto",
		Version: "2.49.0",
	}

	// MatplotlibDistribution ...
	MatplotlibDistribution = Distribution{
		Name:    "matplotlib",
		Version: "3.1.0",
	}

	// GoogleDistribution ...
	GoogleDistribution = Distribution{
		Name:    "google",
		Version: "0",
	}

	// PandasDistribution ...
	PandasDistribution = Distribution{
		Name:    "pandas",
		Version: "0.24.2",
	}
)

// DistributionConstants are a list of defined constants for distributions
var DistributionConstants = []Distribution{
	BuiltinDistribution3,
	NumpyDistribution,
	RequestsDistribution,
	TensorflowDistribution,
	BotoDistribution,
	MatplotlibDistribution,
	GoogleDistribution,
}
