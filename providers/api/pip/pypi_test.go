package pip

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"
)

func TestPyPiNewClientMethod(t *testing.T) {
	pypi := NewPyPiClient(nil, nil)
	if pypi.httpClient != http.DefaultClient {
		t.Errorf("default httpClient is not set on NewPyPiClient instance")
	}
	if pypi.baseUrl != *pyPiBaseURL {
		t.Errorf("default baseURL is not set on NewPyPiClient instance")
	}

	expClient := &http.Client{}
	expUrl, err := url.Parse("http://example.com")
	if err != nil {
		t.Fatalf("unexpected test url parse error: %v", err)
	}
	pypi = NewPyPiClient(expClient, expUrl)
	if pypi.httpClient != expClient {
		t.Errorf("default httpClient is not set on NewPyPiClient instance")
	}
	if pypi.baseUrl != *expUrl {
		t.Errorf("default baseURL is not set on NewPyPiClient instance")
	}
}

func TestPyPiClientPackageMethod(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		expectedPath := "/pypi/package_name/json"
		if r.URL.Path != expectedPath {
			t.Errorf("expected url call is %q, got %q", r.URL.Path, expectedPath)
		}
		_, _ = rw.Write([]byte(sampleProjectJson))
	}))

	expectedObj := PipPackage{}
	err := json.Unmarshal([]byte(sampleProjectJson), &expectedObj)
	if err != nil {
		t.Fatal("testing sampleproject JSON is invalid or structs are broken")
	}

	URL, _ := url.Parse(srv.URL)
	pypi := NewPyPiClient(srv.Client(), URL)
	pkg, err := pypi.Package(context.Background(), "package_name")
	if err != nil {
		t.Fatalf("unexpected Release() error: %v", err)
	}

	if !reflect.DeepEqual(*pkg, expectedObj) {
		t.Error("expected and actual results are not equal")
	}
}

func TestPyPiClientReleaseMethod(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		expectedPath := "/pypi/package_name/package_version/json"
		if r.URL.Path != expectedPath {
			t.Errorf("expected url call is %q, got %q", r.URL.Path, expectedPath)
		}
		_, _ = rw.Write([]byte(sampleProjectJson))
	}))

	expectedObj := PipPackage{}
	err := json.Unmarshal([]byte(sampleProjectJson), &expectedObj)
	if err != nil {
		t.Fatal("testing sampleproject JSON is invalid or structs are broken")
	}

	URL, _ := url.Parse(srv.URL)
	pypi := NewPyPiClient(srv.Client(), URL)
	pkg, err := pypi.Release(context.Background(), "package_name", "package_version")
	if err != nil {
		t.Fatalf("unexpected Release() error: %v", err)
	}

	if !reflect.DeepEqual(*pkg, expectedObj) {
		t.Error("expected and actual results are not equal")
	}
}

func TestPyPiClientRelease_Errors(t *testing.T) {
	notFoundSrv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.WriteHeader(http.StatusNotFound)
		_, _ = rw.Write([]byte("{}"))
	}))
	incorrectSchemaSrv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		_, _ = rw.Write([]byte("hello_world!"))
	}))

	cases := []struct {
		Name    string
		Server  *httptest.Server
		Ctx     context.Context
		PkgName string
		Version string
	}{
		{"", notFoundSrv, context.Background(), "", ""},
		{"", notFoundSrv, nil, "package_name", "version"},
		{"", notFoundSrv, context.Background(), "package_name", "version"},
		{"", incorrectSchemaSrv, context.Background(), "package_name", "version"},
	}

	for _, testCase := range cases {
		t.Run(testCase.Name, func(t *testing.T) {
			URL, _ := url.Parse(testCase.Server.URL)
			pypi := NewPyPiClient(testCase.Server.Client(), URL)

			pkg, err := pypi.Release(testCase.Ctx, testCase.PkgName, testCase.Version)
			if err == nil {
				t.Error("expected error on empty name, got none")
			}
			if pkg != nil {
				t.Error("expected nil PipPackage on incorrect request")
			}
		})
	}
}

var sampleProjectJson = `{
	"info":{
	   "author":"A. Random Developer",
	   "author_email":"author@example.com",
	   "bugtrack_url": "http://example.com",
	   "classifiers":[
		  "Topic :: Software Development :: Build Tools"
	   ],
	   "description":"# A sample Python project\n\n![Python Logo](https://www.python.org/static/community_logos/python-logo.png \"Sample inline image\")\n\nA sample project that exists as an aid to the [Python Packaging User\nGuide][packaging guide]'s [Tutorial on Packaging and Distributing\nProjects][distribution tutorial].\n\nThis project does not aim to cover best practices for Python project\ndevelopment as a whole. For example, it does not provide guidance or tool\nrecommendations for version control, documentation, or testing.\n\n[The source for this project is available here][src].\n\nMost of the configuration for a Python project is done in the 'setup.py' file,\nan example of which is included in this project. You should edit this file\naccordingly to adapt this sample project to your needs.\n\n----\n\nThis is the README file for the project.\n\nThe file should use UTF-8 encoding and can be written using\n[reStructuredText][rst] or [markdown][md use] with the appropriate [key set][md\nuse]. It will be used to generate the project webpage on PyPI and will be\ndisplayed as the project homepage on common code-hosting services, and should be\nwritten for that purpose.\n\nTypical contents for this file would include an overview of the project, basic\nusage examples, etc. Generally, including the project changelog in here is not a\ngood idea, although a simple \u201cWhat's New\u201d section for the most recent version\nmay be appropriate.\n\n[packaging guide]: https://packaging.python.org\n[distribution tutorial]: https://packaging.python.org/tutorials/packaging-projects/\n[src]: https://github.com/pypa/sampleproject\n[rst]: http://docutils.sourceforge.net/rst.html\n[md]: https://tools.ietf.org/html/rfc7764#section-3.5 \"CommonMark variant\"\n[md use]: https://packaging.python.org/specifications/core-metadata/#description-content-type-optional\n\n\n",
	   "description_content_type":"text/markdown",
	   "docs_url":"http://example.com/docs",
	   "download_url":"http://example.com/download",
	   "downloads":{
		  "last_day":-1,
		  "last_month":-1,
		  "last_week":-1
	   },
	   "home_page":"https://github.com/pypa/sampleproject",
	   "keywords":"sample setuptools development",
	   "license":"",
	   "maintainer":"",
	   "maintainer_email":"",
	   "name":"sampleproject",
	   "package_url":"https://pypi.org/project/sampleproject/",
	   "platform":"",
	   "project_url":"https://pypi.org/project/sampleproject/",
	   "project_urls":{
		  "Bug Reports":"https://github.com/pypa/sampleproject/issues",
		  "Funding":"https://donate.pypi.org",
		  "Homepage":"https://github.com/pypa/sampleproject",
		  "Say Thanks!":"http://saythanks.io/to/example",
		  "Source":"https://github.com/pypa/sampleproject/"
	   },
	   "release_url":"https://pypi.org/project/sampleproject/2.0.0/",
	   "requires_dist":[
		  "peppercorn",
		  "check-manifest ; extra == 'dev'",
		  "coverage ; extra == 'test'"
	   ],
	   "requires_python":">=3.5, <4",
	   "summary":"A sample Python project",
	   "version":"2.0.0",
	   "yanked":false,
	   "yanked_reason":null
	},
	"last_serial":7562906,
	"releases":{
	   "1.0":[],
	   "1.2.0":[
		  {
			 "comment_text":"",
			 "digests":{
				"md5":"bab8eb22e6710eddae3c6c7ac3453bd9",
				"sha256":"7a7a8b91086deccc54cac8d631e33f6a0e232ce5775c6be3dc44f86c2154019d"
			 },
			 "downloads":-1,
			 "filename":"sampleproject-1.2.0-py2.py3-none-any.whl",
			 "has_sig":false,
			 "md5_digest":"bab8eb22e6710eddae3c6c7ac3453bd9",
			 "packagetype":"bdist_wheel",
			 "python_version":"2.7",
			 "requires_python":null,
			 "size":3795,
			 "upload_time":"2015-06-14T14:38:05",
			 "upload_time_iso_8601":"2015-06-14T14:38:05.875222Z",
			 "url":"https://files.pythonhosted.org/packages/30/52/547eb3719d0e872bdd6fe3ab60cef92596f95262e925e1943f68f840df88/sampleproject-1.2.0-py2.py3-none-any.whl",
			 "yanked":false,
			 "yanked_reason":null
		  },
		  {
			 "comment_text":"",
			 "digests":{
				"md5":"d3bd605f932b3fb6e91f49be2d6f9479",
				"sha256":"3427a8a5dd0c1e176da48a44efb410875b3973bd9843403a0997e4187c408dc1"
			 },
			 "downloads":-1,
			 "filename":"sampleproject-1.2.0.tar.gz",
			 "has_sig":false,
			 "md5_digest":"d3bd605f932b3fb6e91f49be2d6f9479",
			 "packagetype":"sdist",
			 "python_version":"source",
			 "requires_python":null,
			 "size":3148,
			 "upload_time":"2015-06-14T14:37:56",
			 "upload_time_iso_8601":"2015-06-14T14:37:56.383366Z",
			 "url":"https://files.pythonhosted.org/packages/eb/45/79be82bdeafcecb9dca474cad4003e32ef8e4a0dec6abbd4145ccb02abe1/sampleproject-1.2.0.tar.gz",
			 "yanked":false,
			 "yanked_reason":null
		  }
	   ],
	   "1.3.0":[
		  {
			 "comment_text":"",
			 "digests":{
				"md5":"de98c6cdd6962d67e7368d2f9d9fa934",
				"sha256":"ab855ea282734dd216e8be4a42899a6fa8d2ce8f65b41c6379b69c1f804d6b1c"
			 },
			 "downloads":-1,
			 "filename":"sampleproject-1.3.0-py2.py3-none-any.whl",
			 "has_sig":false,
			 "md5_digest":"de98c6cdd6962d67e7368d2f9d9fa934",
			 "packagetype":"bdist_wheel",
			 "python_version":"py2.py3",
			 "requires_python":">=2.7, !=3.0.*, !=3.1.*, !=3.2.*, !=3.3.*, <4",
			 "size":3988,
			 "upload_time":"2019-05-28T20:23:12",
			 "upload_time_iso_8601":"2019-05-28T20:23:12.721927Z",
			 "url":"https://files.pythonhosted.org/packages/a1/fd/3564a5176430eac106c27eff4de50b58fc916f5083782062cea3141acfaa/sampleproject-1.3.0-py2.py3-none-any.whl",
			 "yanked":false,
			 "yanked_reason":null
		  },
		  {
			 "comment_text":"",
			 "digests":{
				"md5":"3dd8fce5e4e2726f343de4385ec8d479",
				"sha256":"ee67ab9c8b445767203e7d9523d029287f737c60524a3c0e0c36cc504e0f24d7"
			 },
			 "downloads":-1,
			 "filename":"sampleproject-1.3.0.tar.gz",
			 "has_sig":false,
			 "md5_digest":"3dd8fce5e4e2726f343de4385ec8d479",
			 "packagetype":"sdist",
			 "python_version":"source",
			 "requires_python":">=2.7, !=3.0.*, !=3.1.*, !=3.2.*, !=3.3.*, <4",
			 "size":5913,
			 "upload_time":"2019-05-28T20:23:13",
			 "upload_time_iso_8601":"2019-05-28T20:23:13.940627Z",
			 "url":"https://files.pythonhosted.org/packages/a6/aa/0090d487d204f5de30035c00f6c71b53ec7f613138d8653eebac50f47f45/sampleproject-1.3.0.tar.gz",
			 "yanked":false,
			 "yanked_reason":null
		  }
	   ]
	},
	"urls":[
	   {
		  "comment_text":"",
		  "digests":{
			 "md5":"34b3750e8a39e7c2930cac64cd44ca0a",
			 "sha256":"2b0c55537193b792098977fdb62f0acbaeb2c3cfc56d0e24ccab775201462e04"
		  },
		  "downloads":-1,
		  "filename":"sampleproject-2.0.0-py3-none-any.whl",
		  "has_sig":false,
		  "md5_digest":"34b3750e8a39e7c2930cac64cd44ca0a",
		  "packagetype":"bdist_wheel",
		  "python_version":"py3",
		  "requires_python":">=3.5, <4",
		  "size":4209,
		  "upload_time":"2020-06-25T19:09:43",
		  "upload_time_iso_8601":"2020-06-25T19:09:43.103653Z",
		  "url":"https://files.pythonhosted.org/packages/b8/f7/dd9223b39f683690c30f759c876df0944815e47b588cb517e4b9e652bcf7/sampleproject-2.0.0-py3-none-any.whl",
		  "yanked":false,
		  "yanked_reason":null
	   },
	   {
		  "comment_text":"",
		  "digests":{
			 "md5":"7414660845e963b2a0e4d52c6d4a111f",
			 "sha256":"d99de34ffae5515db43916ec47380d3c603e9dead526f96581b48c070cc816d3"
		  },
		  "downloads":-1,
		  "filename":"sampleproject-2.0.0.tar.gz",
		  "has_sig":false,
		  "md5_digest":"7414660845e963b2a0e4d52c6d4a111f",
		  "packagetype":"sdist",
		  "python_version":"source",
		  "requires_python":">=3.5, <4",
		  "size":7298,
		  "upload_time":"2020-06-25T19:09:43",
		  "upload_time_iso_8601":"2020-06-25T19:09:43.925879Z",
		  "url":"https://files.pythonhosted.org/packages/8d/c7/bf2d01f14bc647c4ef2299dec830560a9b55a582ecf9e0e43af740c79ccd/sampleproject-2.0.0.tar.gz",
		  "yanked":false,
		  "yanked_reason":null
	   }
	]
 }`
