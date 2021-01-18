package parsers

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"strings"
	"unicode"

	"github.com/dephub/dephub-core/providers/fetchers"
)

// NewPipParser constructs pip files parser.
// If 'filename' parameter is an empty string - 'requirements.txt' will be used instead.
func NewPipParser(fetcher fetchers.FileFetcher, filename string) DependencyParser {
	if filename == "" {
		return &PipParser{fetcher: fetcher, SourceName: "requirements.txt"}
	}
	return &PipParser{fetcher: fetcher, SourceName: filename}
}

// PipParser represents concrete pip parser implementation.
type PipParser struct {
	fetcher fetchers.FileFetcher
	// SourceName is the source filename (e.g. 'requirements.txt')
	SourceName string
}

// Requirements method always returns nil values, because pip doesnt support fully locked deps lists.
func (c PipParser) Requirements(ctx context.Context) ([]Requirement, error) {
	// There are no locked deps list in pip (like in php or other dep managers)
	// todo: maybe we should use '==' and '===' dependencies from Requirements list? But then
	// it's only a subset, not all dependencies.
	return nil, nil
}

// Constraints method returns python dependencies constraints.
func (c PipParser) Constraints(ctx context.Context) ([]Constraint, error) {
	b, err := c.fetcher.FileContent(ctx, c.SourceName)
	if err != nil {
		if err == fetchers.ErrFileNotFound {
			return nil, ErrFileNotFound
		}
		return nil, fmt.Errorf("unable to fetch python(pip) dependencies from the source: %w", err)
	}

	reqs := parseRequirementsTxt(b)
	res := []Constraint{}

	for k, v := range reqs {
		res = append(res, Constraint{
			Name:    k,
			Version: v,
		})
	}

	return res, nil
}

// parseRequirementsTxt contains requirements.txt files parsing logic.
// TODO: improve add additional signatures support.
func parseRequirementsTxt(fileContent []byte) map[string]string {
	var result map[string]string = make(map[string]string)
	delimeters := []string{"===", "==", ">=", "<=", "<", ">", "~=", "!="}
	scanner := bufio.NewScanner(bytes.NewReader(fileContent))
	for scanner.Scan() {
		if scanner.Text() == "" {
			continue
		}
		line := stripSpaces(scanner.Text()) // remove any spaces
		pkg := line
		version := "*" // default version
		// Ignore unsupported signatures
		if strings.HasPrefix(line, "#") || strings.HasPrefix(line, "-r") || strings.Contains(line, "/") || strings.Contains(line, ";") {
			continue
		}
		line = strings.Split(line, "#")[0] // remove comments

		for _, delim := range delimeters {
			if strings.Contains(line, delim) {
				pkg = strings.Split(line, delim)[0]
				version = delim + strings.Split(line, delim)[1]
				break
			}
		}

		result[pkg] = version
	}

	return result
}

// Fast way to strip all whitespaces from a string
func stripSpaces(str string) string {
	var b strings.Builder
	b.Grow(len(str))
	for _, ch := range str {
		if !unicode.IsSpace(ch) {
			b.WriteRune(ch)
		}
	}
	return b.String()
}
