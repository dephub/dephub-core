/*
Package pip provides a client for using the PyPi pip public API.

Usage:
	todo:
*/
package pip

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"
)

// pyPiBaseURL - PyPi base API url (used as default client baseURL)
var pyPiBaseURL *url.URL

// pyPiHostname - PyPi API hostname (used as default API).
//
// Packagist is the main Composer repository. It aggregates public PHP packages installable with Composer.
// You can get more info on Packagist and it's official API here: packagist.org/apidoc
var pyPiHostname string = "https://pypi.org"

func init() {
	pyPiBaseURL, _ = url.Parse(pyPiHostname)
}

// NewPyPiClient constructs a new PyPiClient
//
// If httpClient or URL is nil - default values will be used.
// Pass URL only if you are sure that the address is compatible with PyPi public API.
func NewPyPiClient(httpClient *http.Client, URL *url.URL) PyPiClient {
	if URL == nil {
		URL = pyPiBaseURL
	}
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return PyPiClient{httpClient: httpClient, baseUrl: *URL}
}

// PyPiClient is used to communicate with PyPi compatible API service.
type PyPiClient struct {
	httpClient *http.Client
	baseUrl    url.URL
}

// Package method is used to get information about packages, their versions and metadata.
//
// This method is almost identical to the 'release' one, so i'm keeping it for
// resemblance with API routes and as a shortut for the Release()
func (pc PyPiClient) Package(ctx context.Context, name string) (*PipPackage, error) {
	return pc.Release(ctx, name, "")
}

// Package method is used to get information about packages, their versions and metadata.
func (pc PyPiClient) Release(ctx context.Context, name, version string) (*PipPackage, error) {
	if name == "" {
		return nil, fmt.Errorf("pacakge name is required and can't be empty")
	}

	var path string
	if version == "" {
		path = fmt.Sprintf("%s/pypi/%s/json", &pc.baseUrl, name)
	} else {
		path = fmt.Sprintf("%s/pypi/%s/%s/json", &pc.baseUrl, name, version)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("unable to create a request: %w", err)
	}
	resp, err := pc.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("unable to send the request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Pypi returned with !=200 status code")
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("unable to read the response body: %w", err)
	}

	pp := PipPackage{}
	if err = json.Unmarshal(body, &pp); err != nil {
		return nil, fmt.Errorf("unable to parse the response body: %w", err)
	}

	return &pp, nil
}

// PipPackage represents full package metadata from PyPi.
type PipPackage struct {
	Info       PipPackageInfo        `json:"info"`
	LastSerial int                   `json:"last_serial"`
	Releases   PipPackageReleaseList `json:"releases"`
	Urls       []struct {
		CommentText string `json:"comment_text"`
		Digests     struct {
			Md5    string `json:"md5"`
			Sha256 string `json:"sha256"`
		} `json:"digests"`
		Downloads         int       `json:"downloads"`
		Filename          string    `json:"filename"`
		HasSig            bool      `json:"has_sig"`
		Md5Digest         string    `json:"md5_digest"`
		Packagetype       string    `json:"packagetype"`
		PythonVersion     string    `json:"python_version"`
		RequiresPython    string    `json:"requires_python"`
		Size              int       `json:"size"`
		UploadTime        string    `json:"upload_time"`
		UploadTimeIso8601 time.Time `json:"upload_time_iso_8601"`
		URL               string    `json:"url"`
		Yanked            bool      `json:"yanked"`
		YankedReason      string    `json:"yanked_reason"`
	} `json:"urls"`
}

// PipPackageInfo represents package information data.
type PipPackageInfo struct {
	Author                 string   `json:"author"`
	AuthorEmail            string   `json:"author_email"`
	BugtrackURL            string   `json:"bugtrack_url"`
	Classifiers            []string `json:"classifiers"`
	Description            string   `json:"description"`
	DescriptionContentType string   `json:"description_content_type"`
	DocsURL                string   `json:"docs_url"`
	DownloadURL            string   `json:"download_url"`
	Downloads              struct {
		LastDay   int `json:"last_day"`
		LastMonth int `json:"last_month"`
		LastWeek  int `json:"last_week"`
	} `json:"downloads"`
	HomePage        string `json:"home_page"`
	Keywords        string `json:"keywords"`
	License         string `json:"license"`
	Maintainer      string `json:"maintainer"`
	MaintainerEmail string `json:"maintainer_email"`
	Name            string `json:"name"`
	PackageURL      string `json:"package_url"`
	Platform        string `json:"platform"`
	ProjectURL      string `json:"project_url"`
	ProjectUrls     struct {
		BugReports string `json:"Bug Reports"`
		Funding    string `json:"Funding"`
		Homepage   string `json:"Homepage"`
		SayThanks  string `json:"Say Thanks!"`
		Source     string `json:"Source"`
	} `json:"project_urls"`
	ReleaseURL     string      `json:"release_url"`
	RequiresDist   []string    `json:"requires_dist"`
	RequiresPython string      `json:"requires_python"`
	Summary        string      `json:"summary"`
	Version        string      `json:"version"`
	Yanked         bool        `json:"yanked"`
	YankedReason   interface{} `json:"yanked_reason"`
}

// PipPackageInfo represents package releases list, where map key is version and value is array of releases.
type PipPackageReleaseList map[string][]PipPackageRelease

// PipPackageRelease represents one concrete release information block.
type PipPackageRelease struct {
	Comment  string `json:"comment_text"`
	Filename string `json:"filename"`
	Digests  struct {
		Md5    string `json:"md5"`
		Sha256 string `json:"sha256"`
	} `json:"digests"`
	Downloads         int       `json:"downloads"`
	HasSig            bool      `json:"has_sig"`
	Md5Digest         string    `json:"md5_digest"`
	Packagetype       string    `json:"packagetype"`
	PythonVersion     string    `json:"python_version"`
	RequiresPython    string    `json:"requires_python"`
	Size              int       `json:"size"`
	UploadTime        string    `json:"upload_time"`
	UploadTimeIso8601 time.Time `json:"upload_time_iso_8601"`
	URL               string    `json:"url"`
	Yanked            bool      `json:"yanked"`
	YankedReason      string    `json:"yanked_reason"`
}
