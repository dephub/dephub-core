package packagist

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"
)

func getTestingClient(t *testing.T, srv *httptest.Server) *PackagistClient {
	t.Helper()
	url, _ := url.Parse(srv.URL)
	cl, err := NewClient(srv.Client(), url)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	return cl
}

// func testTableTest

func TestNewClientMethod(t *testing.T) {
	cl, err := NewClient(nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cl.baseURL.String() != packagistHostname {
		t.Errorf("nil client url is incorrect, expected '%s', got '%s'", packagistHostname, cl.baseURL.String())
	}
	if cl.HttpClient != http.DefaultClient {
		t.Error("nil client is not a default one")
	}
}

func TestNewClient_IncorrectUrl(t *testing.T) {
	packagistHostname = "httz://}oh no{"
	cl, err := NewClient(nil, nil)
	if err == nil {
		t.Errorf("expected incorrect url error, got nothing")
	}
	if cl != nil {
		t.Errorf("expected nil packagist client, got %+v", cl)
	}
}

func TestListMethod(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		expectedUrl := "/packages/list.json?type=package&vendor=testing"
		if r.URL.String() != expectedUrl {
			t.Fatalf("incorrect requested url '%s', expected '%s'", r.URL.String(), expectedUrl)
		}

		rw.Header().Set("Content-Type", "application/json")
		_, _ = rw.Write([]byte(`{
			"packageNames": [
			  "testing/world",
			  "testify/client",
			  "test/testing"
			]
		  }`))
	}))

	expectedResult := &PackagesList{PackageNames: []string{"testing/world", "testify/client", "test/testing"}}

	cl := getTestingClient(t, srv)

	res, _, err := cl.List(context.Background(), &ListOptions{Vendor: "testing", Type: "package"})
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(res, expectedResult) {
		t.Error("unexpected response, structs are not the same")
	}
}

func TestListMethod_Errors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) { _, _ = rw.Write([]byte("Hello world!")) }))
	cl := getTestingClient(t, srv)

	var ctx context.Context
	res, _, err := cl.List(ctx, &ListOptions{Vendor: "testing", Type: "package"})
	if err == nil {
		t.Error("expected error, got nil")
	}
	if res != nil {
		t.Errorf("expected nil result on error, got %-v", res)
	}
}

func TestHttpErrorResponse(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())

	cl := getTestingClient(t, srv)

	req, err := http.NewRequestWithContext(context.Background(), "GET", srv.URL, nil)
	if err != nil {
		t.Errorf("failed to create a response for the test, error returned: %v", err)
	}

	var tst interface{}
	_, err = parseResponse(cl, req, tst)

	if err == nil {
		t.Error("expected 404 error, got none")
	}
}

func TestReqErrorResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) { _, _ = rw.Write([]byte("Hello world!")) }))
	cl := getTestingClient(t, srv)

	req := http.Request{}

	var tst interface{}
	_, err := parseResponse(cl, &req, tst)

	if err == nil {
		t.Error("expected request error, got none")
	}
}

func TestListMethod_EmptyArrayResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Set("Content-Type", "application/json")
		_, _ = rw.Write([]byte(`{
			"packageNames": []
		  }`))
	}))

	cl := getTestingClient(t, srv)

	res, _, err := cl.List(context.Background(), &ListOptions{Vendor: "testing", Type: "package"})
	if err != nil {
		t.Fatal(err)
	}

	if len(res.PackageNames) != 0 {
		t.Error("unexpected response")
	}
}

func TestSearchMethod(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		expectedUrl := "/search.json?page=2&per_page=5&q=testing&tags%5B%5D=f_tag&tags%5B%5D=s_tag&type=package"
		if r.URL.String() != expectedUrl {
			t.Fatalf("incorrect requested url '%s', expected '%s'", r.URL.String(), expectedUrl)
		}

		rw.Header().Set("Content-Type", "application/json")
		_, _ = rw.Write([]byte(`{
			"results" : [
			  {
				"name": "test/world",
				"description": "Test world!",
				"url": "https://example.org/packages/test/world",
				"repository": "https://git.example.org/packages/test/world.git",
				"downloads": 7123,
				"favers": 102
			  },
			  {
				"name": "testing/client",
				"description": "Test client!",
				"url": "https://example.org/packages/testing/client",
				"repository": "https://git.example.org/packages/testing/client.git",
				"downloads": 91842,
				"favers": 3288
			  }
			],
			"total": 2,
			"next": "https://example.org/search.json"
		  }`))
	}))

	expectedResult := &FoundSearch{Results: []FoundPackage{
		{
			Name:        "test/world",
			Description: "Test world!",
			URL:         "https://example.org/packages/test/world",
			Repository:  "https://git.example.org/packages/test/world.git",
			Downloads:   7123,
			Favers:      102,
		},
		{
			Name:        "testing/client",
			Description: "Test client!",
			URL:         "https://example.org/packages/testing/client",
			Repository:  "https://git.example.org/packages/testing/client.git",
			Downloads:   91842,
			Favers:      3288,
		},
	},
		Total:   2,
		NextURL: "https://example.org/search.json",
	}

	cl := getTestingClient(t, srv)

	res, _, err := cl.Search(context.Background(), "testing", &SearchOptions{
		PerPage: 5, Page: 2, Type: "package", Tags: []string{"f_tag", "s_tag"},
	})
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(res.Results, expectedResult.Results) {
		t.Error("unexpected response")
	}
}

func TestSearchMethod_Errors(t *testing.T) {
	// Table test cases
	cases := []struct {
		TestName string
		Ctx      context.Context
		Q        string
	}{
		{"nil opts", context.Background(), ""},
		{"invalid opts", context.Background(), "test"},
		{"nil ctx", nil, "test"},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) { _, _ = rw.Write([]byte("Hello world!")) }))
	cl := getTestingClient(t, srv)

	for _, testData := range cases {
		t.Run(testData.TestName, func(t *testing.T) {
			res, _, err := cl.Search(testData.Ctx, testData.Q, nil)
			if res != nil {
				t.Error("failed response result is not nil")
			}

			if err == nil {
				t.Error(err)
			}
		})
	}
}

func TestSearchMethod_EmptyArrayResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Set("Content-Type", "application/json")
		_, _ = rw.Write([]byte(`{
			"results" : [],
			"total": 0
		  }`))
	}))

	cl := getTestingClient(t, srv)

	res, _, err := cl.Search(context.Background(), "testing", &SearchOptions{
		PerPage: 5, Page: 2, Type: "package", Tags: []string{"f_tag", "s_tag"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Results) != 0 || res.Total != 0 || res.NextURL != "" {
		t.Errorf("expected struct fields, empty results expected, got %+v", res)
	}
}

func TestSearchNextMethod(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		expectedUrl := "/search.json?page=1&q=hello"
		if r.URL.String() != expectedUrl {
			t.Fatalf("incorrect requested url '%s', expected '%s'", r.URL.String(), expectedUrl)
		}
		rw.Header().Set("Content-Type", "application/json")
		_, _ = rw.Write([]byte(`{
			"results" : [
			  {
				"name": "hello/world",
				"description": "Test world!",
				"url": "https://example.org/packages/hello/world",
				"repository": "https://git.example.org/packages/hello/world.git",
				"downloads": 7123,
				"favers": 102
			  }
			],
			"total": 2,
			"next": null
		  }`))
	}))

	cl := getTestingClient(t, srv)

	expectedResult := &FoundSearch{Results: []FoundPackage{{
		Name:        "hello/world",
		Description: "Test world!",
		URL:         "https://example.org/packages/hello/world",
		Repository:  "https://git.example.org/packages/hello/world.git",
		Downloads:   7123,
		Favers:      102,
	}},
		Total:   2,
		NextURL: "",
		client:  *cl,
	}

	foundObj := &FoundSearch{Results: []FoundPackage{},
		Total:   2,
		NextURL: srv.URL + "/search.json?page=1&q=hello",
		client:  *cl,
		q:       "hello",
	}

	ok, err := foundObj.Next(context.Background())
	if err != nil {
		t.Error(err)
	}
	if !ok {
		t.Errorf("next method returned false")
	}

	if !reflect.DeepEqual(foundObj.Results, expectedResult.Results) {
		t.Error("unexpected response")
	}

	if foundObj.NextURL != expectedResult.NextURL || foundObj.Total != expectedResult.Total || foundObj.client != expectedResult.client {
		t.Error("next object")
	}

	ok, err = foundObj.Next(context.Background())
	if err != nil {
		t.Error(err)
	}
	if ok {
		t.Errorf("next method returned false")
	}

	foundObj.NextURL = "http://example.com"
	var ctx context.Context
	ok, err = foundObj.Next(ctx)
	if err == nil {
		t.Error("expected error on nil context, got nil")
	}
	if ok {
		t.Error("expected not ok on nil context, got nil")
	}
}

func TestMetaMethod(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		expectedUrl := "/p/hello/world.json"
		if r.URL.String() != expectedUrl {
			t.Fatalf("incorrect requested url '%s', expected '%s'", r.URL.String(), expectedUrl)
		}

		rw.Header().Set("Content-Type", "application/json")
		_, _ = rw.Write([]byte(`{
			"packages": {
				"joshdifabio/composer": {
					"dev-async": {
						"name": "joshdifabio/composer",
						"description": "Composer helps you declare, manage and install dependencies of PHP projects, ensuring you have the right stack everywhere.",
						"keywords": [
							"package"
						],
						"homepage": "http://getcomposer.org/",
						"version": "dev-async",
						"version_normalized": "dev-async",
						"license": [
							"MIT"
						],
						"authors": [{
								"name": "Nils Adermann",
								"email": "naderman@naderman.de",
								"homepage": "http://www.naderman.de"
							},
							{
								"name": "Jordi Boggiano",
								"email": "j.boggiano@seld.be",
								"homepage": "http://seld.be"
							}
						],
						"source": {
							"type": "git",
							"url": "https://github.com/joshdifabio/composer.git",
							"reference": "40b2acc009d7883003fab85284994c262e78d99e"
						},
						"dist": {
							"type": "zip",
							"url": "https://api.github.com/repos/joshdifabio/composer/zipball/40b2acc009d7883003fab85284994c262e78d99e",
							"reference": "40b2acc009d7883003fab85284994c262e78d99e",
							"shasum": ""
						},
						"type": "library",
						"time": "2015-01-21T14:06:57+00:00",
						"autoload": {
							"psr-0": {
								"Composer": "src/"
							}
						},
						"extra": {
							"branch-alias": {
								"dev-master": "1.0-dev"
							}
						},
						"bin": [
							"bin/composer"
						],
						"require": {
							"php": ">=5.3.2",
							"react/promise": "~1.0|~2.0",
							"react/event-loop": "~0.3.0|~0.4.0|~0.5.0",
							"react/child-process": "~0.3.0|~0.4.0",
							"symfony/console": "~2.3",
							"symfony/finder": "~2.2",
							"symfony/process": "~2.1",
							"justinrainbow/json-schema": "~1.3",
							"seld/jsonlint": "~1.0"
						},
						"require-dev": {
							"phpunit/phpunit": "~4.0"
						},
						"suggest": {
							"ext-zip": "Enabling the zip extension allows you to unzip archives, and allows gzip compression of all internet traffic",
							"ext-openssl": "Enabling the openssl extension allows you to access https URLs for repositories and packages"
						},
						"replace": {
							"composer/composer": "1.0-dev"
						},
						"uid": 311160
					}
				}
			}
		}`))
	}))

	cl := getTestingClient(t, srv)

	want := PackagesMeta{Packages: map[string]PackageMeta{"joshdifabio/composer": {
		VersionMeta{
			Name:              "joshdifabio/composer",
			Description:       "Composer helps you declare, manage and install dependencies of PHP projects, ensuring you have the right stack everywhere.",
			Keywords:          []string{"package"},
			Homepage:          "http://getcomposer.org/",
			Version:           "dev-async",
			VersionNormalized: "dev-async",
			License:           []string{"MIT"},
			Authors: []struct {
				Email string `json:"email"`
				Name  string `json:"name"`
			}{
				{Email: "naderman@naderman.de", Name: "Nils Adermann"},
				{Email: "j.boggiano@seld.be", Name: "Jordi Boggiano"},
			},
		},
	},
	}}

	res, _, err := cl.Meta(context.Background(), "hello", "world")
	if err != nil {
		t.Fatal(err)
	}

	if want.Packages["joshdifabio/composer"][0].Description != res.Packages["joshdifabio/composer"][0].Description {
		t.Error("unexpected struct")
	}
}

func TestMetaMethod_Errors(t *testing.T) {
	// Table test cases
	cases := []struct {
		TestName string
		Ctx      context.Context
		Vendor   string
		Package  string
	}{
		{"nil opts", context.Background(), "", ""},
		{"invalid opts", context.Background(), "incomplete", ""},
		{"nil ctx", nil, "hello", "world"},
		{"valid params, broken api", context.Background(), "hello", "world"},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) { _, _ = rw.Write([]byte("Hello world!")) }))
	cl := getTestingClient(t, srv)

	for _, testData := range cases {
		t.Run(testData.TestName, func(t *testing.T) {
			res, _, err := cl.Meta(testData.Ctx, testData.Vendor, testData.Package)
			if res != nil {
				t.Error("failed response result is not nil")
			}

			if err == nil {
				t.Error(err)
			}
		})
	}
}

func TestDataMethod(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		expectedUrl := "/packages/testing/client.json"
		if r.URL.String() != expectedUrl {
			t.Fatalf("incorrect requested url '%s', expected '%s'", r.URL.String(), expectedUrl)
		}

		rw.Header().Set("Content-Type", "application/json")
		_, _ = rw.Write([]byte(`{
			"package": {
				"name": "testing/client",
				"description": "Composer helps you declare, manage and install dependencies of PHP projects. It ensures you have the right stack everywhere.",
				"time": "2011-10-31T22:21:09+00:00",
				"maintainers": [{
						"name": "Seldaek",
						"avatar_url": "https:\/\/www.gravatar.com\/avatar\/48b79d17cd8a911327cbd88c122b1efb?d=identicon"
					},
					{
						"name": "naderman",
						"avatar_url": "https:\/\/www.gravatar.com\/avatar\/9f580202b05cc640aa9297ab7a1ae764?d=identicon"
					}
				],
				"versions": {
					"dev-master": {
						"name": "composer\/composer",
						"description": "Composer helps you declare, manage and install dependencies of PHP projects. It ensures you have the right stack everywhere.",
						"keywords": [
							"package",
							"dependency",
							"autoload"
						],
						"homepage": "https:\/\/getcomposer.org\/",
						"version": "dev-master",
						"version_normalized": "dev-master",
						"license": [
							"MIT"
						],
						"authors": [{
								"name": "Nils Adermann",
								"email": "naderman@naderman.de",
								"homepage": "https:\/\/www.naderman.de"
							},
							{
								"name": "Jordi Boggiano",
								"email": "j.boggiano@seld.be",
								"homepage": "https:\/\/seld.be"
							}
						],
						"source": {
							"type": "git",
							"url": "https:\/\/github.com\/composer\/composer.git",
							"reference": "a20ee1a448337b8de157110a6cbe6da029a2c669"
						},
						"dist": {
							"type": "zip",
							"url": "https:\/\/api.github.com\/repos\/composer\/composer\/zipball\/a20ee1a448337b8de157110a6cbe6da029a2c669",
							"reference": "a20ee1a448337b8de157110a6cbe6da029a2c669",
							"shasum": ""
						},
						"type": "library",
						"support": {
							"issues": "https:\/\/github.com\/composer\/composer\/issues",
							"irc": "irc:\/\/irc.freenode.org\/composer",
							"source": "https:\/\/github.com\/composer\/composer\/tree\/master"
						},
						"funding": [{
								"url": "https:\/\/packagist.com",
								"type": "custom"
							},
							{
								"url": "https:\/\/github.com\/composer",
								"type": "github"
							},
							{
								"url": "https:\/\/tidelift.com\/funding\/github\/packagist\/composer\/composer",
								"type": "tidelift"
							}
						],
						"time": "2021-01-12T15:31:48+00:00",
						"autoload": {
							"psr-4": {
								"Composer\\": "src\/Composer"
							}
						},
						"extra": {
							"branch-alias": {
								"dev-master": "2.0-dev"
							}
						},
						"bin": [
							"bin\/composer"
						],
						"default-branch": true,
						"require": {
							"php": "^5.3.2 || ^7.0 || ^8.0",
							"composer\/ca-bundle": "^1.0",
							"composer\/semver": "^3.0",
							"composer\/spdx-licenses": "^1.2",
							"composer\/xdebug-handler": "^1.1",
							"justinrainbow\/json-schema": "^5.2.10",
							"psr\/log": "^1.0",
							"seld\/jsonlint": "^1.4",
							"seld\/phar-utils": "^1.0",
							"symfony\/console": "^2.8.52 || ^3.4.35 || ^4.4 || ^5.0",
							"symfony\/filesystem": "^2.8.52 || ^3.4.35 || ^4.4 || ^5.0",
							"symfony\/finder": "^2.8.52 || ^3.4.35 || ^4.4 || ^5.0",
							"symfony\/process": "^2.8.52 || ^3.4.35 || ^4.4 || ^5.0",
							"react\/promise": "^1.2 || ^2.7"
						},
						"require-dev": {
							"symfony\/phpunit-bridge": "^4.2 || ^5.0",
							"phpspec\/prophecy": "^1.10"
						},
						"suggest": {
							"ext-openssl": "Enabling the openssl extension allows you to access https URLs for repositories and packages",
							"ext-zip": "Enabling the zip extension allows you to unzip archives",
							"ext-zlib": "Allow gzip compression of HTTP requests"
						}
					}
				},
				"type": "library",
				"repository": "https:\/\/github.com\/composer\/composer",
				"github_stars": 24591,
				"github_watchers": 618,
				"github_forks": 6002,
				"github_open_issues": 175,
				"language": "PHP",
				"dependents": 2058,
				"suggesters": 11,
				"downloads": {
					"total": 40037835,
					"monthly": 1478411,
					"daily": 85150
				},
				"favers": 24649
			}
		}`))
	}))

	cl := getTestingClient(t, srv)

	res, _, err := cl.Data(context.Background(), "testing", "client")
	if err != nil {
		t.Fatal(err)
	}

	if res.Package.Name != "testing/client" {
		t.Error("unexpected package returned")
	}
}

func TestDataMethod_ErrorResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		expectedUrl := "/packages/testing/client.json"
		if r.URL.String() != expectedUrl {
			t.Fatalf("incorrect requested url '%s', expected '%s'", r.URL.String(), expectedUrl)
		}

		rw.Header().Set("Content-Type", "application/json")
		_, _ = rw.Write([]byte(`{
			"status": "error",
			"message": "Package not found"
		}`))
	}))

	cl := getTestingClient(t, srv)

	_, _, err := cl.Data(context.Background(), "testing", "client")
	if err == nil {
		t.Fatal(err)
	}
}

func TestDataMethod_Errors(t *testing.T) {
	// Table test cases
	cases := []struct {
		TestName string
		Ctx      context.Context
		Vendor   string
		Package  string
	}{
		{"nil opts", context.Background(), "", ""},
		{"invalid opts", context.Background(), "incomplete", ""},
		{"nil ctx", nil, "hello", "world"},
		{"valid params, broken api", context.Background(), "hello", "world"},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) { _, _ = rw.Write([]byte("Hello world!")) }))
	cl := getTestingClient(t, srv)

	for _, testData := range cases {
		t.Run(testData.TestName, func(t *testing.T) {
			res, _, err := cl.Data(testData.Ctx, testData.Vendor, testData.Package)
			if res != nil {
				t.Error("failed response result is not nil")
			}

			if err == nil {
				t.Error(err)
			}
		})
	}
}

func TestStatsMethod(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		expectedUrl := "/statistics.json"
		if r.URL.String() != expectedUrl {
			t.Fatalf("incorrect requested url '%s', expected '%s'", r.URL.String(), expectedUrl)
		}

		rw.Header().Set("Content-Type", "application/json")
		_, _ = rw.Write([]byte(`{
			"totals": {
				"downloads": 32745139730,
				"packages": 295002,
				"versions": 2633514
				}
		}`))
	}))

	expectedStructure := PackagesStats{}
	expectedStructure.Totals.Downloads = 32745139730
	expectedStructure.Totals.Packages = 295002
	expectedStructure.Totals.Versions = 2633514

	cl := getTestingClient(t, srv)

	res, _, err := cl.Stats(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(res.Totals, expectedStructure.Totals) {
		t.Fatal("unexpected result struct data")
	}

	var ctx context.Context
	res, _, err = cl.Stats(ctx)
	if err == nil {
		t.Error("expected error on nil context, got none")
	}
	if res != nil {
		t.Errorf("expected nil result on error, got %+v", res)
	}
}

// Test that SecAdvisor method returns valid structure.
func TestSecAdvisoriesMethod(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		expectedUrl := "/api/security-advisories/?packages%5B%5D=test%2Ftesting"
		if r.URL.String() != expectedUrl {
			t.Fatalf("incorrect requested url '%s', expected '%s'", r.URL.String(), expectedUrl)
		}

		rw.Header().Set("Content-Type", "application/json")
		_, _ = rw.Write([]byte(`{
			"advisories": {
			  "test/testing": [{
				  "packageName": "test/testing",
				  "remoteId": "test/testing/2014-12-29-1.yaml",
				  "title": "Header injection in NativeMailerHandler",
				  "link": "https://github.com/Seldaek/monolog/pull/448#issuecomment-68208704",
				  "cve": null,
				  "affectedVersions": ">=1.8.0,<1.12.0",
				  "source": "FriendsOfPHP/security-advisories",
				  "reportedAt": "2014-12-29 00:00:00",
				  "composerRepository": "https://example.com"
				}
			  ]}}`))
	}))

	expectedStructure := SecAdvisories{Advisories: AdvisoriesContainer{"test/testing": {{
		PackageName:        "test/testing",
		RemoteID:           "test/testing/2014-12-29-1.yaml",
		Title:              "Header injection in NativeMailerHandler",
		Link:               "https://github.com/Seldaek/monolog/pull/448#issuecomment-68208704",
		Cve:                "",
		AffectedVersions:   ">=1.8.0,<1.12.0",
		Source:             "FriendsOfPHP/security-advisories",
		ReportedAt:         "2014-12-29 00:00:00",
		ComposerRepository: "https://example.com",
	}}}}

	cl := getTestingClient(t, srv)

	res, _, err := cl.SecAdvisories(context.Background(), []string{"test/testing"})
	if err != nil {
		t.Fatal(err)
	}

	if len(res.Advisories) != len(expectedStructure.Advisories) {
		t.Error("incorrect 'advisories' count")
	}

	if !reflect.DeepEqual(res.Advisories, expectedStructure.Advisories) {
		t.Fatal("unexpected result struct data")
	}
}

func TestSecAdvisoriesMethod_EmptyArrayResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Set("Content-Type", "application/json")
		_, _ = rw.Write([]byte(`{
			"advisories": []
			}`))
	}))

	cl := getTestingClient(t, srv)

	res, _, err := cl.SecAdvisories(context.Background(), []string{"test/testing"})
	if err != nil {
		t.Fatal(err)
	}

	if len(res.Advisories) != 0 {
		t.Errorf("expected empty advisories, got %d", len(res.Advisories))
	}
}

func TestSecAdvisories_Errors(t *testing.T) {
	// Table test cases
	cases := []struct {
		TestName string
		Ctx      context.Context
		Packages []string
	}{
		{"nil opts", context.Background(), nil},
		{"invalid opts", context.Background(), []string{}},
		{"nil ctx", nil, []string{"correct"}},
		{"valid params, broken api", context.Background(), []string{"correct"}},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) { _, _ = rw.Write([]byte("Hello world!")) }))
	cl := getTestingClient(t, srv)

	for _, testData := range cases {
		t.Run(testData.TestName, func(t *testing.T) {
			res, _, err := cl.SecAdvisories(testData.Ctx, testData.Packages)
			if res != nil {
				t.Error("failed response result is not nil")
			}

			if err == nil {
				t.Error(err)
			}
		})
	}
}

func TestAffectedVersionsNormalizeMethod(t *testing.T) {
	issue := Advisory{AffectedVersions: ">=6,<6.2.1|>=4.0.0-rc2,<4.2.4|>=5,<5.3.1"}
	expected := ">=6,<6.2.1||>=4.0.0-rc2,<4.2.4||>=5,<5.3.1"
	normalized := issue.AffectedVersionsNormalized()
	if normalized != expected {
		t.Errorf("affected versions normalization is incorrect, expected '%s', got '%s", expected, normalized)
	}

	issue = Advisory{AffectedVersions: ">=6,<6.2.1"}
	expected = ">=6,<6.2.1"
	normalized = issue.AffectedVersionsNormalized()
	if normalized != expected {
		t.Errorf("affected versions normalization is incorrect, expected '%s', got '%s", expected, normalized)
	}
}
