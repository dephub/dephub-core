package versioneer

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

/*
Composer versions and constraints semantic parsing implementation.
*/

// composerOprFunc represents composer constraint operator check function.
// It returns true if the version is satisfied by the constraint.
type composerOprFunc func(v Version, c composerConstraint) bool

// composerConfig is used to store composer parser configuration.
type composerConfig struct {
	operators              map[string]composerOprFunc // List of supported constraints operators mapped to check functions (e.g. '>=')
	versionRgx             string                     // Composer version regexp (e.g. v1.2.3-hello)
	wildcardRgx            string                     // Composer wildcard version regexp (e.g. v1.2.*-dev)
	constraintsRgxCompiled *regexp.Regexp             // Compiled composer constraint+wildcard regexp
	versionRgxCompiled     *regexp.Regexp             // Compiled version regexp
}

// composerCfg is a global composer parser configuration.
var composerCfg composerConfig

// wildcard position marks (e.g. 0 for '1.*.3')
const (
	wildcardNone = iota - 1
	wildcardMajor
	wildcardMinor
	wildcardPatch
)

// Composer parser config initialization and expressions compiling.
func init() {
	composerCfg.versionRgx = `v?([0-9]+)(\.[0-9]+)?(\.[0-9]+)?(-([0-9A-Za-z\-]+(\.[0-9A-Za-z\-]+)*))?(\+([0-9A-Za-z\-]+(\.[0-9A-Za-z\-]+)*))?`
	composerCfg.wildcardRgx = `v?([0-9|x|X|\*]+)(\.[0-9|x|X|\*]+)?(\.[0-9|x|X|\*]+)?(-([0-9A-Za-z\-]+(\.[0-9A-Za-z\-]+)*))?(\+([0-9A-Za-z\-]+(\.[0-9A-Za-z\-]+)*))?`
	// Supported composer constraints operators
	composerCfg.operators = map[string]composerOprFunc{
		"":   composerConstraintEqual,
		"!=": composerConstraintNotEqual,
		">":  composerConstraintGreaterThan,
		"<":  composerConstraintLessThan,
		">=": composerConstraintGreaterThanEqual,
		"<=": composerConstraintLessThanEqual,
		"~":  composerConstraintTilde,
		"^":  composerConstraintCaret,
	}

	// Convert all existing convertion options into escaped regex words
	ops := make([]string, 0, len(composerCfg.operators))
	for k := range composerCfg.operators {
		ops = append(ops, regexp.QuoteMeta(k))
	}
	composerCfg.constraintsRgxCompiled = regexp.MustCompile(fmt.Sprintf(`^\s*(%s)\s*(%s)\s*$`, strings.Join(ops, "|"), composerCfg.wildcardRgx))
	composerCfg.versionRgxCompiled = regexp.MustCompile("^" + composerCfg.versionRgx + "$")
}

func composerConstraintEqual(v Version, c composerConstraint) bool {
	switch c.wildcard {
	case wildcardNone:
		return v.Major() == c.ver.Major() && v.Minor() == c.ver.Minor() && v.Patch() == c.ver.Patch() // fully equal
	case wildcardMajor:
		return true // * is always equal to any version
	case wildcardMinor:
		return v.Major() == c.ver.Major() // major equal
	case wildcardPatch:
		return v.Major() == c.ver.Major() && v.Minor() == c.ver.Minor() // major equal, minor equal
	}
	return false
}

func composerConstraintNotEqual(v Version, c composerConstraint) bool {
	return !composerConstraintEqual(v, c)
}

func composerConstraintGreaterThan(v Version, c composerConstraint) bool {
	// No wildcard needed
	switch true {
	case v.Major() != c.ver.Major():
		return v.Major() > c.ver.Major()
	case v.Minor() != c.ver.Minor():
		return v.Minor() > c.ver.Minor()
	case v.Patch() != c.ver.Patch():
		return v.Patch() > c.ver.Patch()
	}
	return false
}

func composerConstraintLessThan(v Version, c composerConstraint) bool {
	// No wildcard needed
	switch true {
	case v.Major() != c.ver.Major():
		return v.Major() < c.ver.Major()
	case v.Minor() != c.ver.Minor():
		return v.Minor() < c.ver.Minor()
	case v.Patch() != c.ver.Patch():
		return v.Patch() < c.ver.Patch()
	}

	return false
}

func composerConstraintGreaterThanEqual(v Version, c composerConstraint) bool {
	return composerConstraintEqual(v, c) || composerConstraintGreaterThan(v, c)
}

func composerConstraintLessThanEqual(v Version, c composerConstraint) bool {
	return composerConstraintEqual(v, c) || composerConstraintLessThan(v, c)
}

func composerConstraintTilde(v Version, c composerConstraint) bool {
	// Return false on less versions.
	if composerConstraintLessThan(v, c) {
		return false
	}

	if c.wildcard == wildcardNone && v.Major() == c.ver.Major() && c.ver.Minor() < v.Minor()+1 {
		return true
	}
	if c.wildcard == wildcardPatch && v.Major() == c.ver.Major() && c.ver.Major() < v.Major()+1 {
		return true
	}
	// '~0.0.0' is a special case, it's basically '*'
	if c.wildcard == wildcardMajor || (c.ver.Major() == 0 && c.ver.Minor() == 0 && c.ver.Patch() == 0) {
		return true
	}

	return false
}

func composerConstraintCaret(v Version, c composerConstraint) bool {
	if composerConstraintLessThan(v, c) {
		return false
	}

	if c.wildcard == wildcardPatch && v.Major() == c.ver.Major() {
		return v.Minor() == c.ver.Minor()
	}

	return composerConstraintTilde(v, c)
}

// NewComposerVersion constructs ready-to-use composer Version instance.
func NewComposerVersion(value string) (Version, error) {
	nval := strings.ToLower(value)
	matches := composerCfg.versionRgxCompiled.FindStringSubmatch(nval)
	if matches == nil {
		return nil, fmt.Errorf("version '%s' is not supported", value)
	}

	var temp int64
	var err error
	if temp, err = strconv.ParseInt(matches[1], 10, 0); err != nil {
		return nil, fmt.Errorf("segment parse error: %s", err)
	}
	sv := ComposerVersion{value: value}
	sv.major = int(temp)
	if matches[2] != "" {
		if temp, err = strconv.ParseInt(strings.TrimPrefix(matches[2], "."), 10, 0); err != nil {
			return nil, fmt.Errorf("segment parse error: %s", err)
		}
		sv.minor = int(temp)
	}
	if matches[3] != "" {
		if temp, err = strconv.ParseInt(strings.TrimPrefix(matches[3], "."), 10, 0); err != nil {
			return nil, fmt.Errorf("segment parse error: %s", err)
		}
		sv.patch = int(temp)
	}

	return sv, nil
}

// NewComposerConstraints constructs ready-to-use composer Constraints instance.
func NewComposerConstraints(value string) (Constraints, error) {
	orsRaw := strings.Split(value, "||")
	ors := make([][]composerConstraint, len(orsRaw))
	for k, v := range orsRaw {
		// https://getcomposer.org/doc/articles/versions.md#version-range
		cs := strings.FieldsFunc(v, func(r rune) bool { return r == ',' || r == ' ' })
		result := make([]composerConstraint, len(cs))
		for i, s := range cs {
			pc, err := parseComposerConstraint(s)
			if err != nil {
				return nil, err
			}
			result[i] = *pc
		}
		ors[k] = result
	}
	return ComposerConstraints{value: value, constraints: ors}, nil
}

// parseComposerConstraint is a utility function to convert raw string unary constraint into composerConstraint.
func parseComposerConstraint(c string) (*composerConstraint, error) {
	matches := composerCfg.constraintsRgxCompiled.FindStringSubmatch(c)
	if matches == nil {
		return nil, fmt.Errorf("constraint not supported: %q", c)
	}

	var (
		operator            = matches[1] // comparison operator from unary constraint string (e.g. '>=')
		rawVersion          = matches[2]
		version             = rawVersion
		wildcard            = wildcardNone
		major, minor, patch = matches[3], strings.TrimPrefix(matches[4], "."), strings.TrimPrefix(matches[5], ".")
	)

	// Mark constrait as wildcard if we encounter any and then normalize raw version to fixed one
	if major == "*" {
		version = "0.0.0"
		wildcard = wildcardMajor
	} else if minor == "*" || minor == "" {
		version = fmt.Sprintf("%s.0.0%s", major, matches[6])
		wildcard = wildcardMinor
	} else if patch == "*" || patch == "" {
		version = fmt.Sprintf("%s.%s.0%s", major, minor, matches[6])
		wildcard = wildcardPatch
	}

	vrs, err := NewComposerVersion(version)
	if err != nil {
		return nil, fmt.Errorf("unable to parse version: %w", err)
	}

	cc := &composerConstraint{
		compare:  composerCfg.operators[operator],
		operator: operator,
		wildcard: wildcard,
		raw:      rawVersion,
		ver:      vrs,
	}

	return cc, nil
}

// ComposerConstraints represent Constraints implementation for Composer package manager.
type ComposerConstraints struct {
	value       string
	constraints [][]composerConstraint
}

// composerConstraint represent unary constraint (e.g. for '>=1.2||<=7.2' one of the constraints is '<=7.2')
type composerConstraint struct {
	compare  composerOprFunc // func used to compare this constraint with fixed version
	operator string
	raw      string
	ver      Version
	wildcard int // -1 = no wildcard, 0 - major, 1 - minor, 2 - patch
}

// match method checks the version.
func (cct composerConstraint) match(v Version) bool {
	return cct.compare(v, cct)
}

// Match method validates that the version is in constraints.
func (cc ComposerConstraints) Match(ver Version) bool {
	for _, or := range cc.constraints {
		andMatches := true
		for _, and := range or {
			if !and.match(ver) {
				andMatches = false
				break
			}
		}
		if andMatches {
			return true
		}
	}
	return false
}

// Value method returns original unmodified raw value of the constraints.
func (cc ComposerConstraints) Value() string {
	return cc.value
}

// ComposerVersion represent Version implementation for Composer package manager.
type ComposerVersion struct {
	major, minor, patch int
	value               string
}

// Value method returns original unmodified raw value of the constraints.
func (cv ComposerVersion) Value() string {
	return cv.value
}

// Match method validates that the version is in constraints.
func (cv ComposerVersion) Match(b Constraints) bool {
	return b.Match(cv)
}

// Major method returns integer value of the major version segment (e.g. '?.0.0')
func (cv ComposerVersion) Major() int {
	return cv.major
}

// Major method returns integer value of the minor version segment (e.g. '0.?.0')
func (cv ComposerVersion) Minor() int {
	return cv.minor
}

// Major method returns integer value of the patch version segment (e.g. '0.0.?')
func (cv ComposerVersion) Patch() int {
	return cv.patch
}
