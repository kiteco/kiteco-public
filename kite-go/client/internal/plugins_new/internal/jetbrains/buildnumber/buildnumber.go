// Package buildnumber represents the JetBrains build number system.
// http://www.jetbrains.org/intellij/sdk/docs/basics/getting_started/build_number_ranges.html
package buildnumber

import (
	"fmt"
	"regexp"
	"strconv"
)

// BuildNumber represents a multi-part build number.
type BuildNumber struct {
	ProductID string
	Branch    int
	Build     int
	Remainder string

	minimumBranch  int
	minimumVersion string
}

var (
	buildMatcher = regexp.MustCompile(`(.+)-(\d+)\.(\d+)(.*)`)

	// product codes are used to locate the plugin dir,
	// e.g. "$HOME/.IdeaIC/config/plugins" or "$HOME/Library/Application Support/PyCHarmCE"
	productCodes = map[string]string{
		"IC":  "IdeaIC",       // IntelliJ Community
		"IE":  "IdeaIE",       // IntelliJ Education
		"IU":  "IntelliJIdea", // IntelliJ Ultimate
		"PC":  "PyCharmCE",
		"PY":  "PyCharm",
		"PE":  "PyCharmEdu",
		"PYA": "PyCharm",
		"GO":  "GoLand",
		"PCA": "PyCharmCE",
		"WS":  "WebStorm",
		"PS":  "PhpStorm",
		"RD":  "Rider",
		"CL":  "CLion",
		"RM":  "RubyMine",
		"OC":  "AppCode",
		"AI":  "AndroidStudio",
	}
)

func invalidBuild(build string) error {
	return fmt.Errorf("unable to parse build information from '%s'", build)
}

// FromString parses the build information from the contents of a build.txt file.
func FromString(buildString string) (BuildNumber, error) {
	groups := buildMatcher.FindStringSubmatch(buildString)
	if len(groups) < 5 {
		return BuildNumber{}, invalidBuild(buildString)
	}
	_, found := productCodes[groups[1]]
	if !found {
		return BuildNumber{}, invalidBuild(buildString)
	}
	branch, _ := strconv.Atoi(groups[2])
	build, _ := strconv.Atoi(groups[3])
	return BuildNumber{
		ProductID:      groups[1],
		Branch:         branch,
		Build:          build,
		Remainder:      groups[4],
		minimumBranch:  193,
		minimumVersion: "2019.3",
	}, nil
}

// CompatibilityMessage checks if this install is compatible with the latest plugin version.
func (n BuildNumber) CompatibilityMessage() string {
	if n.Branch < n.minimumBranch {
		return fmt.Sprintf("build must be %d.0 or higher (found %d.%d)", n.minimumBranch, n.Branch, n.Build)
	}
	return ""
}

// RequiredVersion returns the earliest supported version of the JetBrains IDEs we're supporting
func (n BuildNumber) RequiredVersion() string {
	if n.Branch < n.minimumBranch {
		return n.minimumVersion
	}
	return ""
}

// IsAndroidStudio returns if this is the build of an Android Studio
func (n BuildNumber) IsAndroidStudio() bool {
	return n.ProductID == "AI"
}

func (n BuildNumber) String() string {
	return fmt.Sprintf("%s-%d.%d%s", n.ProductID, n.Branch, n.Build, n.Remainder)
}

// ProductVersion returns a string corresponding to the release name and number, i.e. PyCharmCE2017.1.
// If the build is super old or unknown this will return an empty string.
func (n BuildNumber) ProductVersion() string {
	if n.Branch < 145 {
		// this is an old version that we don't even support with different numbers
		return ""
	}
	p := productCodes[n.ProductID]
	var major int
	var minor int
	// Release 3.5 and previous of PyCharmEdu has an undocumented, arbitrary branch numbering scheme.
	// This hardcodes the 3.5 for 163 because it is the current version.
	// 3.6 will follow the new convention of the JetBrains platform.
	if n.ProductID == "PE" && n.Branch == 163 {
		major = 3
		minor = 5
	} else if n.ProductID == "AI" {
		// special handling for Android Studio
		if n.Branch == 193 {
			return "AndroidStudio4.0"
		}
		if n.Branch == 201 {
			// we don't handle beta builds atm
			return "AndroidStudio4.1"
		}
		if n.Branch == 202 {
			// 2020-10: this is the current canary build
			return "AndroidStudioPreview4.2"
		}
		// unknown builds of Android Studio
		return ""
	} else if n.Branch == 145 {
		major = 2016
		minor = 1
	} else {
		major = 2000 + (n.Branch / 10)
		minor = n.Branch % 10
	}
	return fmt.Sprintf("%s%d.%d", p, major, minor)
}
