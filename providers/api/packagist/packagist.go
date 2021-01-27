/*
Package packagist provides a client for using the Pacakgist public API.

Usage:
	todo:
*/
package packagist

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/go-querystring/query"
)

// packagistHostname - packagist API hostname (used as default API).
//
// Packagist is the main Composer repository. It aggregates public PHP packages installable with Composer.
// You can get more info on Packagist and it's official API here: packagist.org/apidoc
var packagistHostname string = "https://packagist.org"

// PackagistClient is used to send API requests to package repository
type PackagistClient struct {
	baseURL    url.URL
	HttpClient *http.Client
}

// NewClient creates and returns a new client
//
// If a nil URL isprovided, default client is configured for default composer package repository (packagist.org).
// Packagist is the main Composer repository. It aggregates public PHP packages installable with Composer.
// You can get more info on Packagist and it's official API here: packagist.org/apidoc
func NewClient(httpClient *http.Client, URL *url.URL) (*PackagistClient, error) {
	// Generate Packagist.org default client if no URL provided.
	if URL == nil {
		var err error
		if URL, err = url.Parse(packagistHostname); err != nil {
			return nil, err
		}
	}

	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	return &PackagistClient{baseURL: *URL, HttpClient: httpClient}, nil
}

// PackagesList represents list of packages.
type PackagesList struct {
	PackageNames []string `json:"packageNames"`
}

// ListOptions specifies the optional parameters to List() method.
type ListOptions struct {
	// For filtering packages by organization.
	Vendor string `url:"vendor,omitempty"`
	// For filtering packages by type.
	Type string `url:"type,omitempty"`
}

// List method lists all packages from the repository.
//
// ! Calling this method without options will load EVERY
// package from the specified repository, the size of the
// response will usually be huge !
func (c PackagistClient) List(ctx context.Context, opts *ListOptions) (*PackagesList, *http.Response, error) {
	v, err := query.Values(opts)
	if err != nil {
		return nil, nil, fmt.Errorf("error parsing the options: %w", err)
	}

	route := fmt.Sprintf("%s/%s?%s", &c.baseURL, "packages/list.json", v.Encode())
	req, err := http.NewRequestWithContext(ctx, "GET", route, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to create a request: %w", err)
	}

	var pl PackagesList
	var r *http.Response
	if r, err = parseResponse(&c, req, &pl); err != nil {
		return nil, nil, err
	}

	return &pl, r, nil
}

// FoundSearch represents search packages result with mutating(!) pagination logic.
// If you want a concurrent pagination - you should load next pages manually.
type FoundSearch struct {
	Results []FoundPackage `json:"results"`
	Total   int            `json:"total"`
	NextURL string         `json:"next"`
	// Original query used to fetch original data.
	q string
	// Options used to fetch original data.
	opts SearchOptions
	// Client used to fetch original data.
	client PackagistClient
}

// Next loads next page (if exists) and mutates existing struct fields with new data.
// It will use a COPY of the Client used to make initial call to API.
//
// It returns true if the struct is updated and false when there is no pages,
// the second argument is error, it is returned only when there is a fatal
// error with fetching page from API.
func (ps *FoundSearch) Next(ctx context.Context) (bool, error) {
	if ps.NextURL == "" {
		return false, nil
	}
	ps.opts.Page++

	nextPs, _, err := ps.client.Search(ctx, ps.q, &ps.opts)
	if err != nil || ps.NextURL == "" {
		return false, err
	}

	*ps = *nextPs
	return true, nil
}

// FoundPackage is a representation of one package from search result slice.
type FoundPackage struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	URL         string `json:"url"`
	Repository  string `json:"repository"`
	Downloads   int    `json:"downloads"`
	Favers      int    `json:"favers"`
}

// SearchOptions specifies the parameters to Search() method.
type SearchOptions struct {
	// PerPage is used to define the pagination step.
	PerPage int `url:"per_page,omitempty"`
	// Page is used to define page.
	Page int `url:"page,omitempty"`
	// For filtering packages by tags.
	Tags []string `url:"tags,brackets,omitempty"`
	// For filtering packages by type.
	Type string `url:"type,omitempty"`
}

// Search methods is used to search for a specific packages.
func (c PackagistClient) Search(ctx context.Context, q string, opts *SearchOptions) (*FoundSearch, *http.Response, error) {
	if q == "" {
		return nil, nil, fmt.Errorf("'q' option is required for search request")
	}

	v, err := query.Values(opts)
	if err != nil {
		return nil, nil, fmt.Errorf("error parsing the options: %w", err)
	}
	v.Add("q", q)

	route := fmt.Sprintf("%s/%s?%s", &c.baseURL, "search.json", v.Encode())
	req, err := http.NewRequestWithContext(ctx, "GET", route, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to create a request: %w", err)
	}

	pl := FoundSearch{client: c, q: q}
	if opts != nil {
		pl.opts = *opts
	}
	var r *http.Response
	if r, err = parseResponse(&c, req, &pl); err != nil {
		return nil, nil, err
	}

	return &pl, r, nil
}

// PackagesMeta represents meta response object.
type PackagesMeta struct {
	Packages map[string]PackageMeta `json:"packages"`
}

// PackageMeta represents packages container (it contains slice of versions).
//
// Original packagist API returns a map, but, considering all benifits of JSON ordered key->value pairs
// we decided to keep the original order, you can find this key value (from packagist API key response)
// in VersionMeta.Version.
type PackageMeta []VersionMeta

// UnmarshalJSON is used in unmarshalling process to keep the original versions order.
//
// We basically use custom decoder to decode and transform key=>obj values into slice values.
func (pms *PackageMeta) UnmarshalJSON(data []byte) error {
	if len(data) < 1 {
		return fmt.Errorf("invalid slice length %d", len(data))
	}

	d := json.NewDecoder(bytes.NewReader(data))
	t, err := d.Token()
	if err != nil || t != json.Delim('{') {
		return fmt.Errorf("PackageMeta custom unmarshaller failed: %w", err)
	}

	var result []VersionMeta
	for d.More() {
		_, err := d.Token()
		if err != nil {
			return fmt.Errorf("PackageMeta custom unmarshaller failed: %w", err)
		}
		var v VersionMeta
		if err := d.Decode(&v); err != nil {
			return fmt.Errorf("PackageMeta custom unmarshaller failed decoding token: %w", err)
		}
		result = append(result, v)
	}

	*pms = result
	return nil
}

// VersionMeta represents versions container.
type VersionMeta struct {
	Authors []struct {
		Email string `json:"email"`
		Name  string `json:"name"`
	} `json:"authors"`
	Description string `json:"description"`
	Dist        struct {
		Reference string `json:"reference"`
		Shasum    string `json:"shasum"`
		Type      string `json:"type"`
		URL       string `json:"url"`
	} `json:"dist"`
	Homepage string            `json:"homepage"`
	Keywords []string          `json:"keywords"`
	License  []string          `json:"license"`
	Name     string            `json:"name"`
	Replace  map[string]string `json:"replace"`
	Source   struct {
		Reference string `json:"reference"`
		Type      string `json:"type"`
		URL       string `json:"url"`
	} `json:"source"`
	Time              string `json:"time"`
	Type              string `json:"type"`
	UID               int64  `json:"uid"`
	Version           string `json:"version"`
	VersionNormalized string `json:"version_normalized"`
}

// Meta methods is used to search for package metadata.
//
// This API endpoint also contains other packages listed as 'replace' for the main one.
func (c PackagistClient) Meta(ctx context.Context, vendor, pkg string) (*PackagesMeta, *http.Response, error) {
	if vendor == "" || pkg == "" {
		return nil, nil, fmt.Errorf("'package' and 'vendor' options are required for meta request")
	}

	v := url.Values{}
	v.Add("vendor", vendor)
	v.Add("package", pkg)

	route := fmt.Sprintf("%s/p/%s/%s.json", &c.baseURL, v.Get("vendor"), v.Get("package"))
	req, err := http.NewRequestWithContext(ctx, "GET", route, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to create a request: %w", err)
	}

	var pl PackagesMeta
	var r *http.Response
	if r, err = parseResponse(&c, req, &pl); err != nil {
		return nil, nil, err
	}

	return &pl, r, nil
}

// PackageData represents metadata for a package.
// TODO: this response is actually unpredictable (the nature of it is actualy user typed configs)
type PackageData struct {
	Package struct {
		Name        string    `json:"name"`
		Description string    `json:"description"`
		Time        time.Time `json:"time"`
		Maintainers []struct {
			Name      string `json:"name"`
			AvatarURL string `json:"avatar_url"`
		} `json:"maintainers"`

		Versions         map[string]VersionMeta `json:"versions"`
		Type             string                 `json:"type"`
		Repository       string                 `json:"repository"`
		GithubStars      int                    `json:"github_stars"`
		GithubWatchers   int                    `json:"github_watchers"`
		GithubForks      int                    `json:"github_forks"`
		GithubOpenIssues int                    `json:"github_open_issues"`
		Language         string                 `json:"language"`
		Dependents       int                    `json:"dependents"`
		Suggesters       int                    `json:"suggesters"`
		Downloads        struct {
			Total   int `json:"total"`
			Monthly int `json:"monthly"`
			Daily   int `json:"daily"`
		} `json:"downloads"`
		Favers int `json:"favers"`
	} `json:"package"`
}

// Data methods is used to search for package metadata.
//
// This method, in comparison to Meta(), gives you all the infos packagist.org have
// including downloads, dependents count, github info, etc.
//
// Also, it is important to note that the endpoint this method calls is pointing to
// dynamically generated file so for performance reason packagist.org caches the responses
// for twelve hours. As such if the static Meta() is enough use it instead.
func (c PackagistClient) Data(ctx context.Context, vendor, pkg string) (*PackageData, *http.Response, error) {
	if vendor == "" || pkg == "" {
		return nil, nil, fmt.Errorf("'package' and 'vendor' options are required for meta request")
	}

	v := url.Values{}
	v.Add("vendor", vendor)
	v.Add("package", pkg)

	route := fmt.Sprintf("%s/packages/%s/%s.json", &c.baseURL, v.Get("vendor"), v.Get("package"))
	req, err := http.NewRequestWithContext(ctx, "GET", route, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to create a request: %w", err)
	}

	var pl PackageData
	var r *http.Response
	if r, err = parseResponse(&c, req, &pl); err != nil {
		return nil, nil, err
	}

	return &pl, r, nil
}

// PackagesStats represent simple global statistics.
type PackagesStats struct {
	Totals struct {
		Downloads int `json:"downloads"`
		Packages  int `json:"packages"`
		Versions  int `json:"versions"`
	} `json:"totals"`
}

// Stats method returns simple global packagist statistics.
func (c PackagistClient) Stats(ctx context.Context) (*PackagesStats, *http.Response, error) {
	route := fmt.Sprintf("%s/statistics.json", &c.baseURL)
	req, err := http.NewRequestWithContext(ctx, "GET", route, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to create a request: %w", err)
	}

	var pl PackagesStats
	var r *http.Response
	if r, err = parseResponse(&c, req, &pl); err != nil {
		return nil, nil, err
	}

	return &pl, r, nil
}

// SecAdvisories represents security-advisories response.
type SecAdvisories struct {
	Advisories AdvisoriesContainer `json:"advisories,omitempty"`
}

// AdvisoriesContainer contains all the advisories for every package.
type AdvisoriesContainer map[string][]Advisory

// UnmarshalJSON is used to change unmarshalling logic when there is empty array in the response.
//
// Packagist.org returns issues with two formats: '"key":{"k":{...}}' if response has advisories
// and '"key":[]' if there are no issues. Default unmarshaling process will fail, so we should tweak
// it a little to translate empty arrey in response into an empty map.
func (ic *AdvisoriesContainer) UnmarshalJSON(data []byte) error {
	if len(data) < 1 {
		return fmt.Errorf("invalid slice length %d", len(data))
	}
	if data[0] == '[' {
		*ic = AdvisoriesContainer{}
		return nil
	}
	m := map[string][]Advisory{}
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}
	*ic = m
	return nil
}

// Advisory represents a security vulnerability.
type Advisory struct {
	PackageName        string `json:"packageName"`
	RemoteID           string `json:"remoteId"`
	Title              string `json:"title"`
	Link               string `json:"link"`
	Cve                string `json:"cve"`
	AffectedVersions   string `json:"affectedVersions"`
	Source             string `json:"source"`
	ReportedAt         string `json:"reportedAt"`
	ComposerRepository string `json:"composerRepository"`
}

// AffectedVersionsNormalized is used to format semver compatible constraints.
func (si Advisory) AffectedVersionsNormalized() string {
	if !strings.Contains(si.AffectedVersions, "||") && strings.Contains(si.AffectedVersions, "|") {
		return strings.ReplaceAll(si.AffectedVersions, "|", "||")
	}
	return si.AffectedVersions
}

// SecAdvisories method fetches known security vulnerabilities for specified packages.
// Names in packages parameter should be formatted like this: '{vendor}/{package}'.
func (c PackagistClient) SecAdvisories(ctx context.Context, packages []string) (*SecAdvisories, *http.Response, error) {
	if len(packages) == 0 || packages == nil {
		return nil, nil, fmt.Errorf("'packages' parameter must contain at least one package name")
	}

	v := url.Values{}
	for _, pkg := range packages {
		v.Add("packages[]", pkg)
	}

	route := fmt.Sprintf("%s/api/security-advisories/?%s", &c.baseURL, v.Encode())
	req, err := http.NewRequestWithContext(ctx, "GET", route, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to create a request: %w", err)
	}

	var pl SecAdvisories
	var r *http.Response
	if r, err = parseResponse(&c, req, &pl); err != nil {
		return nil, nil, err
	}

	return &pl, r, nil
}

// errorResponse represents packagist error response
type errorResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// parseResponse is used to execute the request and unmarshall the response to dt
func parseResponse(c *PackagistClient, req *http.Request, dt interface{}) (r *http.Response, err error) {
	if r, err = c.HttpClient.Do(req); err != nil {
		return nil, fmt.Errorf("unable to send a request: %w", err)
	}
	defer r.Body.Close()

	if r.StatusCode >= 400 {
		return nil, fmt.Errorf("packagist responded with HTTP error '%d: %s'", r.StatusCode, http.StatusText(r.StatusCode))
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("unable to read response body: %w", err)
	}

	// Handling error responses from packagist api
	var ersp errorResponse
	if perr := json.Unmarshal(body, &ersp); perr == nil && (ersp.Message != "" && ersp.Status != "") {
		return nil, fmt.Errorf("packagist api responded with error '%s'", ersp.Message)
	}

	if err = json.Unmarshal(body, &dt); err != nil {
		return nil, fmt.Errorf("unable to parse response: %w", err)
	}

	return r, nil
}
