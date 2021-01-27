/*
Package dephub provides convinient api for core package management instrumentation and parsing.

Usage:
	todo:
*/
package dephub

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/dephub/dephub-core/providers/fetchers"
	"github.com/dephub/dephub-core/providers/parsers"
)

// todo: add rate limiting error variable

// DepType represents package manager type flag.
type DepType string

// Available package managers
const (
	// ComposerType represents PHP's Composer package manager flag.
	ComposerType = DepType("composer")
	// PIPType represents Python's PIP package manager flag.
	PIPType = DepType("pip")
)

// Constraint represents one dependency/constraint.
type Constraint struct {
	Name    string
	Version string
}

// Requirement represents locked dependency.
type Requirement struct {
	Name    string
	Version string
	// Base indicates if the requirement is base (top level requirement, for example from composer.json)
	Base bool
}

// gitRepoRgx is used to parse repository info from GIT-compatible address string.
//
// Examples matching the regexp:
//     'git@myhostname:vendor/reponame.git'
//     'https://myhostname/vendor/reponame.git' and so on...
// Groups:
//     1: protocol (e.g. 'https://' or 'git@')
//     6: hostname (e.g. 'github.com')
//     8: full repo name (e.g. 'vendor/reponame')
var gitRepoRgx string = `^(((git@)|(git:|ssh:|(http[s]?:\/\/))))([\w\.@\\-~]+)(:|\/)([\w\.@\:\/\-~]+)(\.git)(\/-)?`

// gitRepoRgxCompiled is compiled from gitRepoRgx.
var gitRepoRgxCompiled *regexp.Regexp

func init() {
	gitRepoRgxCompiled = regexp.MustCompile(gitRepoRgx)
}

// DependencySource represents abstraction over package manager source files and
// provides convinient interface to fetch packages information.
type DependencySource interface {
	// Requirements returns list of project's locked dependencies versions (if any).
	Requirements(ctx context.Context, typ DepType) ([]Requirement, error)
	// Constraints returns list of project's dependencies constraints.
	Constraints(ctx context.Context, typ DepType) ([]Constraint, error)
}

func NewMemorySource(files map[string][]byte) DependencySource {
	return &MemoryDependencySource{
		fetchers.ByteMapFetcher{Files: files},
	}
}

type MemoryDependencySource struct {
	fetcher fetchers.ByteMapFetcher
}

// Requirements returns list of project's locked dependencies versions (if any).
//
// Return value is a 'pkg_name:version' map.
func (ldds MemoryDependencySource) Requirements(ctx context.Context, typ DepType) ([]Requirement, error) {
	return parseRequirements(ctx, typ, ldds.fetcher)
}

// Constraints returns list of project's dependencies constraints.
//
// Return value is a 'pkg_name:constraint' map.
func (ldds MemoryDependencySource) Constraints(ctx context.Context, typ DepType) ([]Constraint, error) {
	return parseConstraints(ctx, typ, ldds.fetcher)
}

// gitRepo represents basic repository information.
type gitRepo struct {
	host, vendor, repo string
}

// supGitSrcs - supported git sources.
var supGitSrcs = []string{"github.com"}

// NewGitSource constructs new Git DependencySource implementation.
//
// SHA can both refer to commit hash/branch/tag.
//
// You can pass specific signed httpClient with any information you want the requests go with
// for example you would like to pass OAuth2/BasicAuth information to github API for increased
// rate limits and so on.
//
// repoAddr is your repository address (e.g. 'git@myhostname:vendor/reponame.git')
func NewGitSource(httpClient *http.Client, repoAddr, sha string) (DependencySource, error) {
	repoData, err := parseGitAddr(repoAddr)
	if err != nil {
		return nil, err
	}
	fetcher := fetchers.NewGitHubFetcher(httpClient, repoData.vendor, repoData.repo, sha)
	return &GitDependencySource{fetcher: fetcher}, nil
}

// GitDependencySource represents Git DependencySource implementation,
// capable of communicating with Git repositories and fetching package
// managers specific information from them.
type GitDependencySource struct {
	fetcher fetchers.FileFetcher
}

// Requirements returns list of project's locked dependencies versions (if any).
//
// Return value is a 'pkg_name:version' map.
func (gds GitDependencySource) Requirements(ctx context.Context, typ DepType) ([]Requirement, error) {
	return parseRequirements(ctx, typ, gds.fetcher)
}

// Constraints returns list of project's dependencies constraints.
//
// Return value is a 'pkg_name:constraint' map.
func (gds GitDependencySource) Constraints(ctx context.Context, typ DepType) ([]Constraint, error) {
	return parseConstraints(ctx, typ, gds.fetcher)
}

func parseRequirements(ctx context.Context, typ DepType, fetcher fetchers.FileFetcher) ([]Requirement, error) {
	csts, err := solveParser(typ, fetcher).Requirements(ctx)
	if err != nil {
		return nil, err
	}
	result := []Requirement{}
	for _, cst := range csts {
		result = append(result, Requirement(cst))
	}
	return result, nil
}

func parseConstraints(ctx context.Context, typ DepType, fetcher fetchers.FileFetcher) ([]Constraint, error) {
	csts, err := solveParser(typ, fetcher).Constraints(ctx)
	if err != nil {
		return nil, err
	}
	result := []Constraint{}
	for _, cst := range csts {
		result = append(result, Constraint(cst))
	}
	return result, nil
}

// solveParser - helper to get configured package manager files parser
//
// todo: changable filepaths for parsers
func solveParser(typ DepType, fetcher fetchers.FileFetcher) parsers.DependencyParser {
	var parser parsers.DependencyParser
	switch typ {
	case ComposerType:
		parser = parsers.NewComposerParser(fetcher)
	case PIPType:
		parser = parsers.NewPipParser(fetcher, "")
	}
	return parser
}

// parserGitAddr - helper to parse information from git repository address string
func parseGitAddr(addr string) (*gitRepo, error) {
	matches := gitRepoRgxCompiled.FindStringSubmatch(addr)
	if matches == nil || matches[6] == "" || matches[8] == "" {
		return nil, fmt.Errorf("unsupported git repository format %q", addr)
	}
	hostName, repoName := matches[6], matches[8]

	if !gitHostSupported(hostName) {
		return nil, fmt.Errorf("git source %q is not supported", hostName)
	}

	if !strings.Contains(repoName, "/") {
		return nil, fmt.Errorf("unable to parse vendor from name %q", repoName)
	}
	repoNameParts := strings.Split(repoName, "/")

	return &gitRepo{host: hostName, vendor: repoNameParts[0], repo: repoNameParts[1]}, nil
}

// gitHostSupported - helper to check git source support status
func gitHostSupported(host string) bool {
	for _, v := range supGitSrcs {
		if v == host {
			return true
		}
	}
	return false
}
