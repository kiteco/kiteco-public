package github

// Package is the root struct that stores the stats of a pakcage
type Package struct {
	Name       string
	Counts     int
	Submodules []*Submodule
	Cdf        float64
	Freq       float64
}

// Submodule is the struct that stores the stats of a submodule
type Submodule struct {
	Name    string
	Counts  int
	Methods []*Method
	Cdf     float64
	Freq    float64
}

// Method stores the basic stats for a method of a package.
type Method struct {
	Name  string
	Count int
	Cdf   float64
	Freq  float64
}
