package parsers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/dephub/dephub-core/providers/fetchers"
)

// NewComposerParser constructs Composer files parser.
func NewComposerParser(fetcher fetchers.FileFetcher) DependencyParser {
	return &ComposerParser{fetcher: fetcher}
}

// ComposerParser represents concrete Composer parser implementation.
type ComposerParser struct {
	fetcher fetchers.FileFetcher
}

// ComposerLock represents Composer lock file (composer.lock).
type ComposerLock struct {
	Packages    []Requirement
	PackagesDev []Requirement
}

// ComposerJson represents Composer file (composer.json).
type ComposerJson struct {
	Require    map[string]string
	RequireDev map[string]string
}

// Constraints method returns composer.json constraints.
func (c ComposerParser) Constraints(ctx context.Context) ([]Constraint, error) {
	b, err := c.fetcher.FileContent(ctx, "composer.json")
	if err != nil {
		if err == fetchers.ErrFileNotFound {
			return nil, ErrFileNotFound
		}
		return nil, fmt.Errorf("unable to fetch composer dependencies from the source: %w", err)
	}

	var composer ComposerJson
	err = json.Unmarshal(b, &composer)
	if err != nil {
		return nil, fmt.Errorf("unable to parse composer file content: %w", err)
	}

	res := make([]Constraint, len(composer.Require))

	idx := 0
	for dep, ver := range composer.Require {
		res[idx] = Constraint{
			Name:    dep,
			Version: ver,
		}
		idx++
	}

	return res, nil
}

// Requirements method returns locked packages versions from composer.lock.
func (c ComposerParser) Requirements(ctx context.Context) ([]Requirement, error) {
	b, err := c.fetcher.FileContent(ctx, "composer.lock")
	if err != nil {
		if err == fetchers.ErrFileNotFound {
			return nil, ErrFileNotFound
		}
		return nil, fmt.Errorf("unable to fetch composer dependencies from the source: %w", err)
	}

	var composer ComposerLock
	err = json.Unmarshal(b, &composer)
	if err != nil {
		return nil, fmt.Errorf("unable to parse composer file content: %w", err)
	}

	return composer.Packages, nil
}
