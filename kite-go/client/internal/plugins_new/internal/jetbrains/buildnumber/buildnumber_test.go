package buildnumber

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPyCharmCE_2019(t *testing.T) {
	assert := assert.New(t)
	n, err := FromString("PC-191.3780.4")
	assert.NoError(err)
	assert.Equal("PC", n.ProductID)
	assert.Equal(191, n.Branch)
	assert.Equal(3780, n.Build)
	assert.Equal(".4", n.Remainder)
	assert.Equal("PyCharmCE2019.1", n.ProductVersion())
	assert.Equal("PC-191.3780.4", n.String())
	assert.NotEmpty(n.CompatibilityMessage())
}

func TestPyCharmCE_2017(t *testing.T) {
	assert := assert.New(t)
	n, err := FromString("PC-171.3780.4")
	assert.NoError(err)
	assert.Equal("PC", n.ProductID)
	assert.Equal(171, n.Branch)
	assert.Equal(3780, n.Build)
	assert.Equal(".4", n.Remainder)
	assert.Equal("PyCharmCE2017.1", n.ProductVersion())
	assert.Equal("PC-171.3780.4", n.String())
	assert.NotEmpty(n.CompatibilityMessage())
}

func TestPyCharmCE_2016_1(t *testing.T) {
	assert := assert.New(t)
	n, err := FromString("PC-145.260")
	assert.NoError(err)
	assert.Equal("PC", n.ProductID)
	assert.Equal(145, n.Branch)
	assert.Equal(260, n.Build)
	assert.Equal("", n.Remainder)
	assert.Equal("PyCharmCE2016.1", n.ProductVersion())
	assert.Equal("PC-145.260", n.String())
	assert.NotEmpty(n.CompatibilityMessage())
}

func TestOldVersionHasCompatibilityWarning(t *testing.T) {
	assert := assert.New(t)
	n, err := FromString("IC-143.2370.31")
	assert.NoError(err)
	assert.Equal("IC", n.ProductID)
	assert.Equal(143, n.Branch)
	assert.Equal(2370, n.Build)
	assert.Equal(".31", n.Remainder)
	assert.Empty("", n.ProductVersion())
	assert.Equal("IC-143.2370.31", n.String())
	assert.NotEmpty(n.CompatibilityMessage())
}

func TestIntelliJUltimate(t *testing.T) {
	assert := assert.New(t)
	n, err := FromString("IU-201.94.11.256.42")
	assert.NoError(err)
	assert.Equal("IU", n.ProductID)
	assert.Equal(201, n.Branch)
	assert.Equal(94, n.Build)
	assert.Equal(".11.256.42", n.Remainder)
	assert.Equal("IntelliJIdea2020.1", n.ProductVersion())
	assert.Equal("IU-201.94.11.256.42", n.String())
	assert.Empty(n.CompatibilityMessage())
}

func TestIntelliJEducation(t *testing.T) {
	assert := assert.New(t)
	n, err := FromString("IE-201.94.11.256.42")
	assert.NoError(err)
	assert.Equal("IE", n.ProductID)
	assert.Equal(201, n.Branch)
	assert.Equal(94, n.Build)
	assert.Equal(".11.256.42", n.Remainder)
	assert.Equal("IdeaIE2020.1", n.ProductVersion())
	assert.Equal("IE-201.94.11.256.42", n.String())
	assert.Empty(n.CompatibilityMessage())
}

func TestPyCharmEdu(t *testing.T) {
	assert := assert.New(t)
	n, err := FromString("PE-163.12429")
	assert.NoError(err)
	assert.Equal("PE", n.ProductID)
	assert.Equal(163, n.Branch)
	assert.Equal(12429, n.Build)
	assert.Empty(n.Remainder)
	assert.Equal("PyCharmEdu3.5", n.ProductVersion())
	assert.Equal("PE-163.12429", n.String())
	assert.NotEmpty(n.CompatibilityMessage())
}

func TestPyCharmEdu2020(t *testing.T) {
	assert := assert.New(t)
	n, err := FromString("PE-201.12429")
	assert.NoError(err)
	assert.Equal("PE", n.ProductID)
	assert.Equal(201, n.Branch)
	assert.Equal(12429, n.Build)
	assert.Empty(n.Remainder)
	assert.Equal("PyCharmEdu2020.1", n.ProductVersion())
	assert.Equal("PE-201.12429", n.String())
	assert.Empty(n.CompatibilityMessage())
}

func TestPhpStorm(t *testing.T) {
	assert := assert.New(t)
	n, err := FromString("PS-201.12429")
	assert.NoError(err)
	assert.Equal("PS", n.ProductID)
	assert.Equal(201, n.Branch)
	assert.Equal(12429, n.Build)
	assert.Empty(n.Remainder)
	assert.Equal("PhpStorm2020.1", n.ProductVersion())
	assert.Equal("PS-201.12429", n.String())
	assert.Empty(n.CompatibilityMessage())
}

func TestRider(t *testing.T) {
	assert := assert.New(t)
	n, err := FromString("RD-201.12429")
	assert.NoError(err)
	assert.Equal("RD", n.ProductID)
	assert.Equal(201, n.Branch)
	assert.Equal(12429, n.Build)
	assert.Empty(n.Remainder)
	assert.Equal("Rider2020.1", n.ProductVersion())
	assert.Equal("RD-201.12429", n.String())
	assert.Empty(n.CompatibilityMessage())
}

func TestCLion(t *testing.T) {
	assert := assert.New(t)
	n, err := FromString("CL-201.12429")
	assert.NoError(err)
	assert.Equal("CL", n.ProductID)
	assert.Equal(201, n.Branch)
	assert.Equal(12429, n.Build)
	assert.Empty(n.Remainder)
	assert.Equal("CLion2020.1", n.ProductVersion())
	assert.Equal("CL-201.12429", n.String())
	assert.Empty(n.CompatibilityMessage())
}

func TestRubyMine(t *testing.T) {
	assert := assert.New(t)
	n, err := FromString("RM-201.12429")
	assert.NoError(err)
	assert.Equal("RM", n.ProductID)
	assert.Equal(201, n.Branch)
	assert.Equal(12429, n.Build)
	assert.Empty(n.Remainder)
	assert.Equal("RubyMine2020.1", n.ProductVersion())
	assert.Equal("RM-201.12429", n.String())
	assert.Empty(n.CompatibilityMessage())
}

func TestAppCode(t *testing.T) {
	assert := assert.New(t)
	n, err := FromString("OC-201.12429")
	assert.NoError(err)
	assert.Equal("OC", n.ProductID)
	assert.Equal(201, n.Branch)
	assert.Equal(12429, n.Build)
	assert.Empty(n.Remainder)
	assert.Equal("AppCode2020.1", n.ProductVersion())
	assert.Equal("OC-201.12429", n.String())
	assert.Empty(n.CompatibilityMessage())
}

func TestAndroidStudio193(t *testing.T) {
	assert := assert.New(t)
	n, err := FromString("AI-193.6911.18.40.6821437")
	assert.NoError(err)
	assert.Equal("AI", n.ProductID)
	assert.Equal(193, n.Branch)
	assert.Equal(6911, n.Build)
	assert.EqualValues(".18.40.6821437", n.Remainder)
	assert.Equal("AndroidStudio4.0", n.ProductVersion())
	assert.Equal("AI-193.6911.18.40.6821437", n.String())
	assert.Empty(n.CompatibilityMessage())
}

func TestAndroidStudio201(t *testing.T) {
	assert := assert.New(t)
	n, err := FromString("AI-201.8743.12.41.6823847")
	assert.NoError(err)
	assert.Equal("AI", n.ProductID)
	assert.Equal(201, n.Branch)
	assert.Equal(8743, n.Build)
	assert.EqualValues(".12.41.6823847", n.Remainder)
	assert.Equal("AndroidStudio4.1", n.ProductVersion())
	assert.Equal("AI-201.8743.12.41.6823847", n.String())
	assert.Empty(n.CompatibilityMessage())
}

func TestAndroidStudio202(t *testing.T) {
	assert := assert.New(t)
	n, err := FromString("AI-202.7319.50.42.6863838")
	assert.NoError(err)
	assert.Equal("AI", n.ProductID)
	assert.Equal(202, n.Branch)
	assert.Equal(7319, n.Build)
	assert.EqualValues(".50.42.6863838", n.Remainder)
	assert.Equal("AndroidStudioPreview4.2", n.ProductVersion())
	assert.Equal("AI-202.7319.50.42.6863838", n.String())
	assert.Empty(n.CompatibilityMessage())
}
