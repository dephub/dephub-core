/*
Package fetchers provides file fetching functions for local and remote repositories.

Usage:
	todo:
*/

package fetchers

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/google/go-github/v33/github"
)

var (
	ErrFileNotFound = errors.New("dependency file not found")
)

// FileFetcher interface defines fetchers methods.
type FileFetcher interface {
	FileContent(ctx context.Context, path string) ([]byte, error)
}

// ByteMapFetcher is used for storing file contents in memory (usefull for debugging/testing or for building custom repositories logic)
type ByteMapFetcher struct {
	Files map[string][]byte
}

// FileContent retrieves (if found) []byte contents from it's map using path argument as a key.
func (sf ByteMapFetcher) FileContent(ctx context.Context, path string) ([]byte, error) {
	v, ok := sf.Files[path]
	if !ok {
		return nil, ErrFileNotFound
	}
	return v, nil
}

// GitHubFetcher fetches files from the specified repository.
// Owner and Repo represent '{owner}/{repo}' notation.
// httpClient can be used as OAuth2 or BasicAuth http transport.
type GitHubFetcher struct {
	Owner        string
	Repo         string
	SHA          string
	githubClient *github.Client
}

// NewGitHubFetcher constructs GitHubFileFetcher with specified parameters.
// httpClient can be used as OAuth2 or BasicAuth http transport.
func NewGitHubFetcher(httpClient *http.Client, owner, repo, sha string) FileFetcher {
	return &GitHubFetcher{
		Owner:        owner,
		Repo:         repo,
		SHA:          sha,
		githubClient: github.NewClient(httpClient),
	}
}

// FileContent fetches specified file content from the configured repository.
// Path argument is the root-related file path.
func (p GitHubFetcher) FileContent(ctx context.Context, path string) ([]byte, error) {
	opts := github.RepositoryContentGetOptions{
		Ref: p.SHA,
	}

	rc, dc, resp, err := p.githubClient.Repositories.GetContents(ctx, p.Owner, p.Repo, path, &opts)
	if err != nil {
		if resp.StatusCode == http.StatusNotFound {
			return nil, ErrFileNotFound
		}
		return nil, fmt.Errorf("unable to load '%s' file from github: %w", path, err)
	}

	if len(dc) != 0 {
		return nil, fmt.Errorf("parameter is a directory or not a valid file")
	}

	c, err := rc.GetContent()

	return []byte(c), err
}
