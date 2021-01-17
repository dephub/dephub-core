package ver

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

/*
Pip versions and constraints semantic parsing implementation.
*/

// pipOprFunc represents pip constraint operator check function.
// It returns true if the version is satisfied by the constraint.
type pipOprFunc func(v Version, c pipConstraint) bool

// pipConfig is used to store pip parser configuration.
type pipConfig struct {
	operators              map[string]pipOprFunc // List of supported constraints operators mapped to check functions (e.g. '>=')
	versionRgx             string                // pip version regexp (e.g. v1.2.3-hello)
	wildcardRgx            string                // pip wildcard version regexp (e.g. v1.2.*)
	constraintsRgxCompiled *regexp.Regexp        // Compiled pip constraint+wildcard regexp
	versionRgxCompiled     *regexp.Regexp        // Compiled version regexp
}

// pipCfg is a global pip parser configuration.
var pipCfg pipConfig

// pip parser config initialization and expressions compiling.
func init() {
	pipCfg.versionRgx = `v?([0-9]+)(\.[0-9]+)?(\.[0-9]+)?(\.[0-9]+)?(-([0-9A-Za-z\-]+(\.[0-9A-Za-z\-]+)*))?(\+([0-9A-Za-z\-]+(\.[0-9A-Za-z\-]+)*))?`
	pipCfg.wildcardRgx = `v?([0-9|x|X|\*]+)(\.[0-9|x|X|\*]+)?(\.[0-9|x|X|\*]+)?(-([0-9A-Za-z\-]+(\.[0-9A-Za-z\-]+)*))?(\+([0-9A-Za-z\-]+(\.[0-9A-Za-z\-]+)*))?`
	// Supported pip constraints operators
	pipCfg.operators = map[string]pipOprFunc{
		"":    pipConstraintEqual,
		"==":  pipConstraintEqual,
		"===": pipConstraintArbitraryEqual,
		"!=":  pipConstraintNotEqual,
		">":   pipConstraintGreaterThan,
		"<":   pipConstraintLessThan,
		">=":  pipConstraintGreaterThanEqual,
		"<=":  pipConstraintLessThanEqual,
		"~=":  pipConstraintTildeEqual,
	}

	// Convert all existing convertion options into escaped regex words
	ops := make([]string, 0, len(pipCfg.operators))
	for k := range pipCfg.operators {
		ops = append(ops, regexp.QuoteMeta(k))
	}
	pipCfg.constraintsRgxCompiled = regexp.MustCompile(fmt.Sprintf(`^\s*(%s)\s*(%s)\s*$`, strings.Join(ops, "|"), pipCfg.wildcardRgx))
	pipCfg.versionRgxCompiled = regexp.MustCompile("^" + pipCfg.versionRgx + "$")
}

func pipConstraintEqual(v Version, c pipConstraint) bool {
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

func pipConstraintArbitraryEqual(v Version, c pipConstraint) bool {
	// Arbitrary equality comparisons are simple string equality operations
	// which do not take into account any of the semantic information.
	// !Arbitrary versions and constraints are not fully supported by this package!
	return v.Value() == c.raw
}

func pipConstraintNotEqual(v Version, c pipConstraint) bool {
	return !pipConstraintEqual(v, c)
}

func pipConstraintGreaterThan(v Version, c pipConstraint) bool {
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

func pipConstraintLessThan(v Version, c pipConstraint) bool {
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

func pipConstraintGreaterThanEqual(v Version, c pipConstraint) bool {
	return pipConstraintEqual(v, c) || pipConstraintGreaterThan(v, c)
}

func pipConstraintLessThanEqual(v Version, c pipConstraint) bool {
	return pipConstraintEqual(v, c) || pipConstraintLessThan(v, c)
}

func pipConstraintTildeEqual(v Version, c pipConstraint) bool {
	// Return false on less versions.
	if pipConstraintLessThan(v, c) {
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

// NewPipVersion constructs ready-to-use Pip Version instance.
func NewPipVersion(value string) (Version, error) {
	nval := strings.ToLower(value)
	matches := pipCfg.versionRgxCompiled.FindStringSubmatch(nval)
	if matches == nil {
		return nil, fmt.Errorf("version '%s' is not supported", value)
	}

	var temp int64
	var err error
	if temp, err = strconv.ParseInt(matches[1], 10, 0); err != nil {
		return nil, fmt.Errorf("segment parse error: %s", err)
	}
	sv := PipVersion{value: value}
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

// NewPipConstraints constructs ready-to-use pip Constraints instance.
func NewPipConstraints(value string) (Constraints, error) {
	// https://www.python.org/dev/peps/pep-0440/#version-specifiers
	andsRaw := strings.Split(value, ",")
	ands := make([]pipConstraint, len(andsRaw))
	for k, v := range andsRaw {
		constraint, err := parsePipConstraint(v)
		if err != nil {
			return nil, err
		}
		ands[k] = *constraint
	}
	return PipConstraints{value: value, constraints: ands}, nil
}

// parsePipConstraint is a utility function to convert raw string unary constraint into pipConstraint.
func parsePipConstraint(c string) (*pipConstraint, error) {
	matches := pipCfg.constraintsRgxCompiled.FindStringSubmatch(c)
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

	vrs, err := NewPipVersion(version)
	if err != nil {
		return nil, fmt.Errorf("unable to parse version: %w", err)
	}

	cc := &pipConstraint{
		compare:  pipCfg.operators[operator],
		operator: operator,
		wildcard: wildcard,
		raw:      rawVersion,
		ver:      vrs,
	}

	return cc, nil
}

// PipConstraints represent Constraints implementation for Pip package manager.
type PipConstraints struct {
	value       string
	constraints []pipConstraint
}

// pipConstraint represent unary constraint (e.g. for '>=1.2||<=7.2' one of the constraints is '<=7.2')
type pipConstraint struct {
	compare  pipOprFunc // func used to compare this constraint with fixed version
	operator string
	raw      string
	ver      Version
	wildcard int // -1 = no wildcard, 0 - major, 1 - minor, 2 - patch
}

// match method checks the version.
func (cct pipConstraint) match(v Version) bool {
	return cct.compare(v, cct)
}

// Match method validates that the version is in constraints.
func (cc PipConstraints) Match(ver Version) bool {
	for _, and := range cc.constraints {
		if !and.match(ver) {
			return false
		}
	}
	return true
}

// Value method returns original unmodified raw value of the constraints.
func (cc PipConstraints) Value() string {
	return cc.value
}

// PipVersion represent Version implementation for Pip package manager.
type PipVersion struct {
	major, minor, patch int
	value               string
}

// Value method returns original unmodified raw value of the constraints.
func (cv PipVersion) Value() string {
	return cv.value
}

// Match method validates that the version is in constraints.
func (cv PipVersion) Match(b Constraints) bool {
	return b.Match(cv)
}

// Major method returns integer value of the major version segment (e.g. '?.0.0')
func (cv PipVersion) Major() int {
	return cv.major
}

// Major method returns integer value of the minor version segment (e.g. '0.?.0')
func (cv PipVersion) Minor() int {
	return cv.minor
}

// Major method returns integer value of the patch version segment (e.g. '0.0.?')
func (cv PipVersion) Patch() int {
	return cv.patch
}
