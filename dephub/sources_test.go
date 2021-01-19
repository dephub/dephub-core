package dephub

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/dephub/dephub-core/providers/fetchers"
	"github.com/dephub/dephub-core/providers/parsers"
)

// configureClient configures client that intercepts ALL requests and forwards them into the specified handler.
func configureClient(t *testing.T, handleFunc http.Handler) *http.Client {
	t.Helper()
	srv := httptest.NewTLSServer(handleFunc)

	// Configuring so that all the request go into our handler.
	return &http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, network, _ string) (net.Conn, error) {
				return net.Dial(network, srv.Listener.Addr().String())
			},
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
}

var fileMapMockData = map[string][]byte{
	"composer.json": []byte(`
		{
			"require": {
				"php": ">=7.1.3",
				"barryvdh/laravel-debugbar": "^3.2",
				"cartalyst/sentinel": "2.0.*",
				"davejamesmiller/laravel-breadcrumbs": "^3.0"
			}
		}
	`),
	"composer.lock": []byte(`
		{
			"packages": [
				{
					"name": "aws/aws-sdk-php",
					"version": "3.69.16"
				},
				{
					"name": "barryvdh/laravel-debugbar",
					"version": "v3.2.0"
				},
				{
					"name": "cartalyst/sentinel",
					"version": "v2.0.17"
				}
			]
		}
	`),
	"requirements.txt": []byte(`
			Django==1.11.15
			django-phonenumber-field==1.1.0
			easy-thumbnails==2.4.2
			phonenumberslite==8.2.0
			Pillow==4.3.0
			django-ckeditor==5.3.0
	`),
}

func TestMemoryDependencySource(t *testing.T) {
	depSource := NewMemorySource(fileMapMockData)
	expPipCnsts := map[string]string{
		"Django":                   "==1.11.15",
		"django-phonenumber-field": "==1.1.0",
		"easy-thumbnails":          "==2.4.2",
		"phonenumberslite":         "==8.2.0",
		"Pillow":                   "==4.3.0",
		"django-ckeditor":          "==5.3.0",
	}
	expComposerCnsts := map[string]string{
		"php":                                 ">=7.1.3",
		"barryvdh/laravel-debugbar":           "^3.2",
		"cartalyst/sentinel":                  "2.0.*",
		"davejamesmiller/laravel-breadcrumbs": "^3.0",
	}
	expComposerReqs := map[string]string{
		"aws/aws-sdk-php":           "3.69.16",
		"barryvdh/laravel-debugbar": "v3.2.0",
		"cartalyst/sentinel":        "v2.0.17",
	}

	pipCnsts, err := depSource.Constraints(context.Background(), PIPType)
	if err != nil {
		t.Fatalf("unexpected error on pip memory source constraints: %v", err)
	}
	composerCnsts, err := depSource.Constraints(context.Background(), ComposerType)
	if err != nil {
		t.Fatalf("unexpected error on composer memory source constraints: %v", err)
	}
	pipReqs, err := depSource.Requirements(context.Background(), PIPType)
	if err != nil {
		t.Fatalf("unexpected error on pip memory source requirements: %v", err)
	}
	composerReqs, err := depSource.Requirements(context.Background(), ComposerType)
	if err != nil {
		t.Fatalf("unexpected error on composer memory source requirements: %v", err)
	}

	if !reflect.DeepEqual(pipCnsts, expPipCnsts) {
		t.Errorf("unexpected pip constraints from mem source: %+v", pipCnsts)
	}
	if !reflect.DeepEqual(composerCnsts, expComposerCnsts) {
		t.Errorf("unexpected composer constraints from mem source: %+v", composerCnsts)
	}
	if !reflect.DeepEqual(composerReqs, expComposerReqs) {
		t.Errorf("unexpected composer requirements from mem source: %+v", composerReqs)
	}
	if len(pipReqs) != 0 {
		t.Errorf("expected empty result from pip requirements from mem source, got: %+v", pipReqs)
	}
}

func TestMemoryDependencySource_SourceErrors(t *testing.T) {
	depSource := NewMemorySource(map[string][]byte{})
	resCnsts, err := depSource.Constraints(context.Background(), ComposerType)
	if err == nil || err != parsers.ErrFileNotFound {
		t.Error("expected no file error from empty source, got none")
	}
	if len(resCnsts) != 0 {
		t.Errorf("expected empty result from source with error, got: %+v", resCnsts)
	}

	resReqs, err := depSource.Requirements(context.Background(), ComposerType)
	if err == nil || err != parsers.ErrFileNotFound {
		t.Error("expected no file error from empty source, got none")
	}
	if len(resReqs) != 0 {
		t.Errorf("expected empty result from source with error, got: %+v", resReqs)
	}
}

func TestGitDependencySource_Constructor(t *testing.T) {
	cl := configureClient(t, http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		t.Fatalf("unexpected call to server on git source construction")
		_, _ = rw.Write([]byte("Dont call me >:(!"))
	}))

	depSource, err := NewGitSource(cl, "git@github.com/hello/world.git", "")
	if err != nil {
		t.Errorf("unexpected error on new git source: %v", err)
	}
	if depSource == nil {
		t.Error("expected not nil DependencySource from git source constructor, got nil")
	}
}

func TestGitDependencySource_Constructor_AddrErrors(t *testing.T) {
	testCases := []struct {
		Name          string
		RepoName      string
		ExpectedError string
	}{
		{"", "github.com/hello/world.git", `unsupported git repository format "github.com/hello/world.git"`},
		{"", "git@notgithub.com/hello/world.git", `git source "notgithub.com" is not supported`},
		{"", "http://github.com/hello_world.git", `unable to parse vendor from name "hello_world"`},
	}

	cl := configureClient(t, http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		t.Fatalf("unexpected call to server on git source construction")
		_, _ = rw.Write([]byte("Dont call me >:(!"))
	}))

	for _, cs := range testCases {
		t.Run(cs.Name, func(t *testing.T) {
			depSource, err := NewGitSource(cl, cs.RepoName, "")
			if err == nil || err.Error() != cs.ExpectedError {
				t.Errorf("expected error on invalid git repo addr, got none")
			}
			if depSource != nil {
				t.Errorf("expected nil DependencySource from git source constructor, got: %+v", depSource)
			}
		})
	}
}

func TestGitDependencySource_Methods(t *testing.T) {
	gitDepSource := GitDependencySource{fetcher: fetchers.ByteMapFetcher{Files: fileMapMockData}}

	expPipCnsts := map[string]string{
		"Django":                   "==1.11.15",
		"django-phonenumber-field": "==1.1.0",
		"easy-thumbnails":          "==2.4.2",
		"phonenumberslite":         "==8.2.0",
		"Pillow":                   "==4.3.0",
		"django-ckeditor":          "==5.3.0",
	}
	expComposerCnsts := map[string]string{
		"php":                                 ">=7.1.3",
		"barryvdh/laravel-debugbar":           "^3.2",
		"cartalyst/sentinel":                  "2.0.*",
		"davejamesmiller/laravel-breadcrumbs": "^3.0",
	}
	expComposerReqs := map[string]string{
		"aws/aws-sdk-php":           "3.69.16",
		"barryvdh/laravel-debugbar": "v3.2.0",
		"cartalyst/sentinel":        "v2.0.17",
	}

	pipCnsts, err := gitDepSource.Constraints(context.Background(), PIPType)
	if err != nil {
		t.Fatalf("unexpected error on pip memory source constraints: %v", err)
	}
	composerCnsts, err := gitDepSource.Constraints(context.Background(), ComposerType)
	if err != nil {
		t.Fatalf("unexpected error on composer memory source constraints: %v", err)
	}
	pipReqs, err := gitDepSource.Requirements(context.Background(), PIPType)
	if err != nil {
		t.Fatalf("unexpected error on pip memory source requirements: %v", err)
	}
	composerReqs, err := gitDepSource.Requirements(context.Background(), ComposerType)
	if err != nil {
		t.Fatalf("unexpected error on composer memory source requirements: %v", err)
	}

	if !reflect.DeepEqual(pipCnsts, expPipCnsts) {
		t.Errorf("unexpected pip constraints from mem source: %+v", pipCnsts)
	}
	if !reflect.DeepEqual(composerCnsts, expComposerCnsts) {
		t.Errorf("unexpected composer constraints from mem source: %+v", composerCnsts)
	}
	if !reflect.DeepEqual(composerReqs, expComposerReqs) {
		t.Errorf("unexpected composer requirements from mem source: %+v", composerReqs)
	}
	if len(pipReqs) != 0 {
		t.Errorf("expected empty result from pip requirements from mem source, got: %+v", pipReqs)
	}
}
