/*
Package versioneer provides parsers for every supported dependency management versions and constraints.

Usage:
	todo:
*/
package versioneer

// Version represents a fixed version (e.g. '1.0.3' or 'v3.2', depending on the implementation)
type Version interface {
	Match(b Constraints) bool // Match method validates that the version is in constraints.
	Major() int               // Major method returns integer value of the major version segment (e.g. '?.0.0')
	Minor() int               // Major method returns integer value of the minor version segment (e.g. '0.?.0')
	Patch() int               // Major method returns integer value of the patch version segment (e.g. '0.0.?')
	Value() string            // Value method returns original unmodified raw value of the version.
}

// Constraints represent a constraint definition (e.g. '>=7.2||7.*' depending on the implementation)
type Constraints interface {
	Match(b Version) bool // Match method validates that the version is in constraints.
	Value() string        // Value method returns original unmodified raw value of the constraints.
}
