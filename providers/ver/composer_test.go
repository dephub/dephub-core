package ver

import (
	"fmt"
	"testing"
)

func TestComposerVersion_Parts(t *testing.T) {
	raw := "v1.2.3"
	version, err := NewComposerVersion(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if version.Major() != 1 || version.Minor() != 2 || version.Patch() != 3 || version.Value() != raw {
		t.Errorf("version '%q' parsed incorrectly, got '%+v'", raw, version)
	}
}

func TestComposerVersion_Error(t *testing.T) {
	version, err := NewComposerVersion("hi1.2.3")
	if err == nil {
		t.Error("expected error on invalid version, got none")
	}
	if version != nil {
		t.Errorf("expected nil version on error, got '%+v'", version)
	}
}

func TestComposerConstraints_Parts(t *testing.T) {
	raw := ">=1.2.3||<=1.4.0,  !=1.2.17"
	constr, err := NewComposerConstraints(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if constr.Value() != raw {
		t.Fatalf("unexpected constraint value, expected '%q', got %q", raw, constr.Value())
	}
}

func TestComposerConstraints_Error(t *testing.T) {
	constr, err := NewComposerConstraints(">=1.2.3|<=1.4.0")
	if err == nil {
		t.Error("expected error on invalid constraint, got none")
	}
	if constr != nil {
		t.Errorf("expected nil version on error, got '%+v'", constr)
	}
}

func TestComposerConstraintsAndVersion_MatchMethod(t *testing.T) {
	// Table test
	cases := []struct {
		Constraint string
		Version    string
		Result     bool
	}{
		{">=v1.2.3,<=1.4.0||98.1.*", "1.2.3", true},
		{">=1.2.3,<=1.4.0||98.1.*", "1.3.2", true},
		{">=1.2.3,<=1.4.0||v98.1.*", "v98.1.376", true},
		{">=1.2.3,<=1.4.0||98.1.*", "v98.2.3", false},
		{">=1.2.3,<=v1.4.0||98.1.*", "98.2", false},
		// Equals, wildcards
		{"3.*", "3.0", true},
		{"3.*", "3.17.0", true},
		{"3.*", "3", true},
		{"3.7", "3.7", true},
		{"3.7", "3.7.0", true},
		{"3", "3.7.0", true},
		{"3", "3.7", true},
		{"3", "3", true},
		{"*", "3", true},
		{"v3", "3.7.0", true},
		// Not equals
		{"!=3.7", "3.7", false},
		{"!=3.7", "3.7.0", false},
		{"!=3.7||3.7", "3.7.0", true},
		{"!=3.7,3.7", "3.7.0", false},
		{"!=3.7 3.7", "3.7.0", false},
		{"!=3.7    3.7", "3.7.0", false},
		// Simple comparison tests (>,<)
		{"<3.7", "3.6.0", true},
		{"<3.7.5", "3.7.4", true},
		{"<3.7.5", "3.7.5", false},
		{"<3.7.5", "3.7.6", false},
		{">3.7", "3.7.0", false},
		{">3.7", "3.7.1", true},
		{">3.7", "3.8.0", true},
		{">3.7.5", "3.7.6", true},
		{">3.7.5", "3.7.5", false},
		{">3.7.5", "3.7.4", false},
		// Tilda tests (~)
		{"~1.2", "1.2", true},
		{"~1.2", "1.2.0", true},
		{"~1.2", "1.2.1", true},
		{"~1.2", "1.8.99", true},
		{"~1.2", "1.1", false},
		{"~1.2", "2.0.0", false},
		{"~1.2", "2.1.0", false},
		{"~1.2.3", "1.2.3", true},
		{"~1.2.3", "1.2.199", true},
		{"~1.2.3", "2.0.0", false},
		{"~0.0.0", "123.213.213", true},
		{"~0.0.0", "0.0.0", true},
		{"~*", "123.213.213", true},
		// Caret tests(^)
		{"^1.2.3", "1.2.3", true},
		{"^1.2.3", "1.1.3", false},
		{"^1.2.3", "1.2.8", true},
		{"^1.2.3", "1.8.3", true},
		{"^1.2.3", "2.2.3", false},
		{"^1.2.3", "1.9.3", true},
		{"^0.3", "0.5.0", false},
		{"^0.3", "0.3.9", true},
	}

	for _, tcase := range cases {
		caseName := fmt.Sprintf("%q->%q)", tcase.Version, tcase.Constraint)
		t.Run(caseName, func(t *testing.T) {
			raw := tcase.Constraint
			constr, err := NewComposerConstraints(raw)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if constr.Value() != raw {
				t.Fatalf("unexpected constraint value, expected '%q', got %q", raw, constr.Value())
			}

			ver, err := NewComposerVersion(tcase.Version) // correct
			if err != nil {
				t.Fatalf("unexpected error on version creation: %v", err)
			}
			if constr.Match(ver) != tcase.Result {
				t.Errorf("incorrect constraints(%q)->version(%q) match result, expected '%t', got '%t'", tcase.Constraint, tcase.Version, tcase.Result, !tcase.Result)
			}
			if ver.Match(constr) != tcase.Result {
				t.Errorf("incorrect version(%q)->constraints(%q) match result, expected '%t', got '%t'", tcase.Version, tcase.Constraint, tcase.Result, !tcase.Result)
			}
		})
	}
}
