module github.com/kiteco/kiteco/linux

go 1.15

replace github.com/kiteco/kiteco/kite-go/client/datadeps => ../kite-go/client/datadeps

replace github.com/kiteco/kiteco => ../

require (
	github.com/dustin/go-humanize v1.0.0
	github.com/kiteco/go-bsdiff/v2 v2.0.1 // indirect
	github.com/kiteco/kiteco v0.0.0-00010101000000-000000000000
	github.com/klauspost/cpuid v1.3.1
	github.com/mitchellh/cli v1.1.2
	github.com/rollbar/rollbar-go v1.2.0
	github.com/shirou/gopsutil v2.20.2+incompatible
	github.com/stretchr/testify v1.6.1
	golang.org/x/sys v0.0.0-20201201145000-ef89a241ccb3
)
