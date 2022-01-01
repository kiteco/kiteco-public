package main

import (
	"testing"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythondocs"
)

// This test set uses HTML files stored in the /test directory. All links in
// those files have been converted to "noop" scheme, so as to avoid live URL
// visits.

const (
	RTDDatafile = "./test/TestRTDDatafile.json.gz"
)

func init() {
	rtdProjectNameMap = make(map[string]*pythondocs.RTDProjectInfo)
	rtdProjectHostMap = make(map[string]*pythondocs.RTDProjectInfo)
	loadRTDDatafile(RTDDatafile)
}

func TestCanonicalizeURL_FullURLNoRef(t *testing.T) {
	URL := "http://example.com/test/index.html"
	refURL := ""
	exp := URL
	cURL, err := canonicalizeURL(URL, refURL)
	if err != nil {
		t.Error(err)
	}
	act := cURL.String()
	if act != exp {
		t.Errorf("%s did not match expected %s\n", act, exp)
	}
}

func TestCanonicalizeURL_NoURLFullRef(t *testing.T) {
	URL := ""
	refURL := "http://example.com/test2/another.html"
	exp := refURL
	cURL, err := canonicalizeURL(URL, refURL)
	if err != nil {
		t.Error(err)
	}
	act := cURL.String()
	if act != exp {
		t.Errorf("%s did not match expected %s\n", act, exp)
	}
}

func TestCanonicalizeURL_FullURLFullRef(t *testing.T) {
	URL := "http://example.com/test/index.html"
	refURL := "http://example.com/test2/another.html"
	exp := URL
	cURL, err := canonicalizeURL(URL, refURL)
	if err != nil {
		t.Error(err)
	}
	act := cURL.String()
	if act != exp {
		t.Errorf("%s did not match expected %s\n", act, exp)
	}
}

func TestCanonicalizeURL_PartialURLFullRef(t *testing.T) {
	URL := "/test/index.html"
	refURL := "http://example.com/test2/another.html"
	exp := "http://example.com/test/index.html"
	cURL, err := canonicalizeURL(URL, refURL)
	if err != nil {
		t.Error(err)
	}
	act := cURL.String()
	if act != exp {
		t.Errorf("%s did not match expected %s\n", act, exp)
	}
}

func TestCanonicalizeURL_PartialLocalURLFullRef(t *testing.T) {
	URL := "./test/index.html"
	refURL := "http://example.com/test2/another.html"
	exp := "http://example.com/test2/test/index.html"
	cURL, err := canonicalizeURL(URL, refURL)
	if err != nil {
		t.Error(err)
	}
	act := cURL.String()
	if act != exp {
		t.Errorf("%s did not match expected %s\n", act, exp)
	}
}

func TestCanonicalizeURL_FileURLFullRef(t *testing.T) {
	URL := "index.html"
	refURL := "http://example.com/test2/another.html"
	exp := "http://example.com/test2/index.html"
	cURL, err := canonicalizeURL(URL, refURL)
	if err != nil {
		t.Error(err)
	}
	act := cURL.String()
	if act != exp {
		t.Errorf("%s did not match expected %s\n", act, exp)
	}
}

func TestCanonicalizeURL_JustAnchorURLFullRef(t *testing.T) {
	URL := "#doc"
	refURL := "http://example.com/test2/another.html"
	exp := refURL
	cURL, err := canonicalizeURL(URL, refURL)
	if err != nil {
		t.Error(err)
	}
	act := cURL.String()
	if act != exp {
		t.Errorf("%s did not match expected %s\n", act, exp)
	}
}

func TestCanonicalizeURL_PartialAnchorURLFullRef(t *testing.T) {
	URL := "/test/index.html#doc"
	refURL := "http://example.com/test2/another.html"
	exp := "http://example.com/test/index.html"
	cURL, err := canonicalizeURL(URL, refURL)
	if err != nil {
		t.Error(err)
	}
	act := cURL.String()
	if act != exp {
		t.Errorf("%s did not match expected %s\n", act, exp)
	}
}

func TestCanonicalizeURL_PartialLocalAnchorURLFullRef(t *testing.T) {
	URL := "./test/index.html#doc"
	refURL := "http://example.com/test2/another.html"
	exp := "http://example.com/test2/test/index.html"
	cURL, err := canonicalizeURL(URL, refURL)
	if err != nil {
		t.Error(err)
	}
	act := cURL.String()
	if act != exp {
		t.Errorf("%s did not match expected %s\n", act, exp)
	}
}

func TestCanonicalizeURL_FullAnchorURLFullRef(t *testing.T) {
	URL := "http://example.com/test/index.html#doc"
	refURL := "http://example.com/test2/another.html"
	exp := "http://example.com/test/index.html"
	cURL, err := canonicalizeURL(URL, refURL)
	if err != nil {
		t.Error(err)
	}
	act := cURL.String()
	if act != exp {
		t.Errorf("%s did not match expected %s\n", act, exp)
	}
}

func TestDocsURL_ReadTheDocsEntry(t *testing.T) {
	exp := "http://virtualenv.readthedocs.org/"
	pName := "virtualenv"
	pd := pythondocs.NewPackageDescriptor(pName)
	rtde := checkReadTheDocsEntryByName(pName)
	if rtde == nil {
		t.Errorf("Expected ReadTheDocs entry for %s\n", pName)
	}
	pd.SetDocsURL(rtde.RTDSubdomain)
	act := pd.DocsURL
	if act != exp {
		t.Errorf("%s did not match expected %s\n", act, exp)
	}
}

func TestDocsURL_PyPIStructuredFooterTable(t *testing.T) {
	URL := "file://./test/PyPIStructuredFooterTable.html"
	exp := "noop://example.com/distribute/table"
	pd := pythondocs.NewPackageDescriptor("")
	err := parsePyPIPackagePage(URL, pd)
	if err != nil {
		t.Errorf("Error parsing PyPI page (%s): %s\n", URL, err)
	}
	act := pd.DocsURL
	if act != exp {
		t.Errorf("%s did not match expected %s\n", act, exp)
	}
}

func TestDocsURL_PyPIWithinLinkText(t *testing.T) {
	URL := "file://./test/PyPIWithinLinkText.html"
	exp := "noop://example.com//celery/within"
	pd := pythondocs.NewPackageDescriptor("")
	err := parsePyPIPackagePage(URL, pd)
	if err != nil {
		t.Errorf("Error parsing PyPI page (%s): %s\n", URL, err)
	}
	act := pd.DocsURL
	if act != exp {
		t.Errorf("%s did not match expected %s\n", act, exp)
	}
}

func TestDocsURL_PyPIWithinLinkLeadingText(t *testing.T) {
	URL := "file://./test/PyPIWithinLinkLeadingText.html"
	exp := "noop://example.com//distribute/leading"
	pd := pythondocs.NewPackageDescriptor("")
	err := parsePyPIPackagePage(URL, pd)
	if err != nil {
		t.Errorf("Error parsing PyPI page (%s): %s\n", URL, err)
	}
	act := pd.DocsURL
	if act != exp {
		t.Errorf("%s did not match expected %s\n", act, exp)
	}
}

func TestDocsURL_OtherCanonicalURL(t *testing.T) {
	URL := "file://./test/OtherCanonicalURL.html"
	exp := "noop://celery.readthedocs.org/en/latest/"
	pd := pythondocs.NewPackageDescriptor("")
	err := parseOtherPage(URL, pd)
	if err != nil {
		t.Errorf("Error parsing other page (%s): %s\n", URL, err)
	}
	act := pd.DocsURL
	if act != exp {
		t.Errorf("%s did not match expected %s\n", act, exp)
	}
}

func TestDocsURL_OtherRTDLink(t *testing.T) {
	URL := "file://./test/OtherRTDLink.html"
	exp := "noop://celery.readthedocs.org/en/latest/"
	pd := pythondocs.NewPackageDescriptor("")
	err := parseOtherPage(URL, pd)
	if err != nil {
		t.Errorf("Error parsing other page (%s): %s\n", URL, err)
	}
	act := pd.DocsURL
	if act != exp {
		t.Errorf("%s did not match expected %s\n", act, exp)
	}
}

func TestDocsURL_OtherWithinLinkText(t *testing.T) {
	URL := "file://./test/OtherWithinLinkText.html"
	exp := "noop://docs.celeryproject.org/en/master/within"
	pd := pythondocs.NewPackageDescriptor("")
	err := parseOtherPage(URL, pd)
	if err != nil {
		t.Errorf("Error parsing other page (%s): %s\n", URL, err)
	}
	act := pd.DocsURL
	if act != exp {
		t.Errorf("%s did not match expected %s\n", act, exp)
	}
}

func TestDocsURL_OtherWithinLinkLeadingText(t *testing.T) {
	URL := "file://./test/OtherWithinLinkLeadingText.html"
	exp := "noop://docs.celeryproject.org/en/master/leading"
	pd := pythondocs.NewPackageDescriptor("")
	err := parseOtherPage(URL, pd)
	if err != nil {
		t.Errorf("Error parsing other page (%s): %s\n", URL, err)
	}
	act := pd.DocsURL
	if act != exp {
		t.Errorf("%s did not match expected %s\n", act, exp)
	}
}

func TestDocsURL_GitHubRTDLink(t *testing.T) {
	URL := "file://./test/GitHubRTDLink.html"
	exp := "noop://requests.readthedocs.org/"
	pd := pythondocs.NewPackageDescriptor("requests")
	err := parseGitHubPage(URL, pd)
	if err != nil {
		t.Errorf("Error parsing GitHub page (%s): %s\n", URL, err)
	}
	act := pd.DocsURL
	if act != exp {
		t.Errorf("%s did not match expected %s\n", act, exp)
	}
}

func TestDocsURL_GitHubWithinLinkText(t *testing.T) {
	URL := "file://./test/GitHubWithinLinkText.html"
	exp := "noop://docs.python-requests.org/within"
	pd := pythondocs.NewPackageDescriptor("requests")
	err := parseGitHubPage(URL, pd)
	if err != nil {
		t.Errorf("Error parsing GitHub page (%s): %s\n", URL, err)
	}
	act := pd.DocsURL
	if act != exp {
		t.Errorf("%s did not match expected %s\n", act, exp)
	}
}

func TestDocsURL_GitHubWithinLinkLeadingText(t *testing.T) {
	URL := "file://./test/GitHubWithinLinkLeadingText.html"
	exp := "noop://docs.python-requests.org/leading"
	pd := pythondocs.NewPackageDescriptor("requests")
	err := parseGitHubPage(URL, pd)
	if err != nil {
		t.Errorf("Error parsing GitHub page (%s): %s\n", URL, err)
	}
	act := pd.DocsURL
	if act != exp {
		t.Errorf("%s did not match expected %s\n", act, exp)
	}
}
