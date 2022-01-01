package pythondocs

const (
	// DefaultRTDDatafilePath is the default location for the RTD info file.
	DefaultRTDDatafilePath = "/var/kite/data/docs/python/rtdinfo.json.gz"
	// DefaultDocsDatafilePath is the default location for the docs info file.
	DefaultDocsDatafilePath = "/var/kite/data/docs/python/docsinfo.json.gz"
	// DefaultDocsSourcePath is the default location for the documentation source files.
	DefaultDocsSourcePath = "/var/kite/data/docs/python/source/"
	// DefaultDocsParsedPath is the default location for the parsed documentation tuple files.
	DefaultDocsParsedPath = "/var/kite/data/docs/python/parsed/"
)

// RTDProjectInfo holds the info about a project hosted on readtheDocs.org.
type RTDProjectInfo struct {
	AbsoluteURL         string        `json:"absolute_url,omitempty"`        // URI for project within RTD.
	AllowComments       bool          `json:"allow_comments,omitempty"`      // Whether to allow comments on the project page.
	AnalyticsCode       string        `json:"analytics_code,omitempty"`      // Analytics tracking code.
	CanonicalURL        string        `json:"canonical_url,omitempty"`       // The canonical URL for the project.
	CommentModeration   bool          `json:"comment_moderation,omitempty"`  // Whether comments are moderated.
	SphinxConfPyFile    string        `json:"conf_py_file,omitempty"`        // Sphinx configuration file.
	Copyright           string        `json:"copyright,omitempty"`           // Copyright information.
	CrateURL            string        `json:"crate_url,omitempty"`           // Crate.io URL.
	DefaultBranch       string        `json:"default_branch,omitempty"`      // Default branch.
	DefaultVersion      string        `json:"default_version,omitempty"`     // Default version.
	Description         string        `json:"description,omitempty"`         // Description of project.
	DjangoPackagesURL   string        `json:"django_packages_url,omitempty"` // Djangopackages.com URI.
	DocumentationType   string        `json:"documentation_type,omitempty"`  // Either "sphinx" or "sphinx_htmldir".
	DownloadFormatURLs  DocFormatURLs `json:"downloads,omitempty"`           // Download links for various formats of the docs.
	ID                  int           `json:"id,omitempty"`                  // Project ID.
	Language            string        `json:"language,omitempty"`            // Project language.
	Mirror              bool          `json:"mirror,omitempty"`              // Whether there is a mirror for the docs?
	ModifiedDate        string        `json:"modified_date,omitempty"`       // Last modified date.
	Name                string        `json:"name,omitempty"`                // Project name.
	NumMajor            int           `json:"num_major,omitempty"`           // The major version number.
	NumMinor            int           `json:"num_minor,omitempty"`           // The minor version number.
	NumPoint            int           `json:"num_point,omitempty"`           // The point version number.
	PrivacyLevel        string        `json:"privacy_level,omitempty"`       // The privacy level of the project.
	ProjectURL          string        `json:"project_url,omitempty"`         // Project homepage.
	PublishedDate       string        `json:"pub_date,omitempty"`            // Last published date.
	PythonInterpreter   string        `json:"python_interpreter,omitempty"`  // Python interpreter to use.
	RepoURI             string        `json:"repo,omitempty"`                // URI for VCS repository.
	RepoType            string        `json:"repo_type,omitempty"`           // Type of VCS repository (e.g., "git").
	PIPRequirementsFile string        `json:"requirements_file,omitempty"`   // Pip requirements file for building docs.
	ResourceURI         string        `json:"resource_uri,omitempty"`        // URI for project object from API.
	SingleVersion       bool          `json:"single_version,omitempty"`      // Whether there is only one version.
	Skip                bool          `json:"skip,omitempty"`                // Unknown.
	Slug                string        `json:"slug,omitempty"`                // Slug.
	RTDSubdomain        string        `json:"subdomain,omitempty"`           // Project subdomain on readthedocs (e.g., http://pip.readthedocs.org).
	DocfileSuffix       string        `json:"suffix,omitempty"`              // File suffix of docfiles (usually ".rst").
	SphinxTheme         string        `json:"theme,omitempty"`               // Sphinx theme.
	UseSystemPackages   bool          `json:"use_system_packages,omitempty"` // Whether to use system packages.
	UseVirtualEnv       bool          `json:"use_virtualenv,omitempty"`      // Whether to build the project in a virtualenv.
	AdminURIs           []string      `json:"users,omitempty"`               // readthedocs.org user URIs for project administrators.
	Version             string        `json:"version,omitempty"`             // Deprecated. Always empty.
	VersionPrivacyLevel string        `json:"version_privacy_level,omitempty"`
}

// DocFormatURLs is a field in RTDProjectInfo that holds download URLs for
// various file formats of the documentation.
type DocFormatURLs struct {
	EPub    string `json:"epub,omitempty"`
	HTMLZip string `json:"htmlzip,omitempty"`
	PDF     string `json:"pdf,omitempty"`
}

// --

// RTDProjectsAPIResponse holds the response from the readthedocs projects API
// (i.e., http://readthedocs.org/api/v1/project/).
type RTDProjectsAPIResponse struct {
	Meta    RTDMetaData      `json:"meta"`
	Objects []RTDProjectInfo `json:"objects"`
}

// RTDMetaData holds the metadata in RTDProjectsAPIResponse that describes the
// response. In particular, it can be used to drive further queries to the API.
type RTDMetaData struct {
	Limit      int    `json:"limit"`       // Number of projects returned
	Next       string `json:"next"`        // URI for next set of projects
	Offset     int    `json:"offset"`      // Current offset used for pagination
	Previous   string `json:"previous"`    // URI for previous set of projects
	TotalCount int    `json:"total_count"` // Total number of projects
}

// --

// PypiEntry holds the information obtained from a package's PyPI page.
type PypiEntry struct {
	Name          string   `json:"Name,omitempty"`
	Summary       string   `json:"Summary,omitempty"`
	Rank          string   `json:"Rank,omitempty"`
	DownloadCount int      `json:"DownloadCount,omitempty"`
	HomepageURL   string   `json:"HomepageURL,omitempty"`
	DocsURLs      []string `json:"DocsURLs,omitempty"`
	OtherURLs     []string `json:"OtherURLs,omitempty"`
}

// PackageDescriptor contains knowledge about a Python package.
type PackageDescriptor struct {
	Name             string          `json:"Name"`
	DocsURL          string          `json:"DocsURL,omitempty"` // The first docs URL found.
	PyPIEntry        *PypiEntry      `json:"PyPIEntry,omitempty"`
	ReadTheDocsEntry *RTDProjectInfo `json:"ReadTheDocsEntry,omitempty"`
}

// NewPackageDescriptor returns a pointer to a newly initialized PackageDescriptor.
func NewPackageDescriptor(name string) *PackageDescriptor {
	return &PackageDescriptor{
		Name:      name,
		PyPIEntry: &PypiEntry{},
	}
}

// TrySetDocsURL sets the documentation URL only if it has not already been set.
func (pd *PackageDescriptor) TrySetDocsURL(URL string) bool {
	if pd.DocsURL == "" {
		pd.SetDocsURL(URL)
		return true
	}
	pd.PyPIEntry.DocsURLs = append(pd.PyPIEntry.DocsURLs, URL)
	return false
}

// SetDocsURL sets the documentation URL.
func (pd *PackageDescriptor) SetDocsURL(URL string) {
	pd.DocsURL = URL
	pd.PyPIEntry.DocsURLs = append(pd.PyPIEntry.DocsURLs, URL)
}
