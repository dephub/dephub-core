/*
Package parsers provides parsers for every supported dependency management fixture files.

Goals:
 - Parsing requirements file into readable struct

Usage:
	todo:
*/
package parsers

import (
	"context"
	"errors"
)

var (
	ErrFileNotFound = errors.New("file not found")
)

// DependencyParser represents basic interface for parsers in this package.
type DependencyParser interface {
	// Requirements have to return list of locked dependencies (if not possible - return nills)
	Requirements(context.Context) ([]Requirement, error)
	// Constraints have to return list of dependencies (with constraints or not).
	// These dependencies do not represent locked ones.
	Constraints(context.Context) ([]Constraint, error)
}

// Constraint represents one dependency/constraint.
// TODO: add normalization logic to translate constraints from different parsers into semver compatible form
type Constraint struct {
	Name    string
	Version string
}

// Requirement represents locked dependency.
type Requirement struct {
	Name    string
	Version string
}
