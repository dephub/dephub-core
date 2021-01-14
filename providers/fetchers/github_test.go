package fetchers

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
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

func TestFetchContentMethod(t *testing.T) {
	cl := configureClient(t, http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		_, _ = rw.Write([]byte(`{
			"content" : "Hello world!"
		}`))
	}))

	expected := "Hello world!"

	fetcher := NewGitHubFetcher(cl, "test", "testing", "")
	content, err := fetcher.FileContent(context.Background(), "test.txt")
	if err != nil {
		t.Error(err)
	}
	if string(content) != expected {
		t.Errorf("expected content '%s', got '%s'", expected, string(content))
	}
}

func TestFetchContentMethod_HttpNotFound(t *testing.T) {
	cl := configureClient(t, http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.WriteHeader(http.StatusNotFound)
		_, _ = rw.Write([]byte(`{
			"message": "Not Found",
			"documentation_url": "https://docs.github.com/rest/reference/repos#get-repository-content"
		  }`))
	}))

	fetcher := NewGitHubFetcher(cl, "test", "testing", "")
	_, err := fetcher.FileContent(context.Background(), "test.txt")
	if err == nil {
		t.Error(err)
	}
}

func TestFetchContentMethod_MultipleReposError(t *testing.T) {
	cl := configureClient(t, http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		_, _ = rw.Write([]byte(`[
			{
			  "name": "CODE_OF_CONDUCT.md",
			  "path": ".github/CODE_OF_CONDUCT.md",
			  "sha": "2b4a5fccdaf12f98cf8e255affa28cfd7e6a784d",
			  "url": "https://api.github.com/repos/golang/go/contents/.github/CODE_OF_CONDUCT.md?ref=master"
			},
			{
			  "name": "ISSUE_TEMPLATE",
			  "path": ".github/ISSUE_TEMPLATE",
			  "sha": "5cbfc09fe76804461d5bf2221d8a6e5ceff5c385",
			  "url": "https://api.github.com/repos/golang/go/contents/.github/ISSUE_TEMPLATE?ref=master"
			}
		  ]`))
	}))

	fetcher := NewGitHubFetcher(cl, "test", "testing", "")
	_, err := fetcher.FileContent(context.Background(), "test.txt")
	if err == nil && err.Error() != "parameter is a directory or not a valid file" {
		t.Error(err)
	}
}
