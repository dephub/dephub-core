package ver

import (
	"fmt"
	"testing"
)

func TestPipVersion_Parts(t *testing.T) {
	raw := "v1.2.3"
	version, err := NewPipVersion(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if version.Major() != 1 || version.Minor() != 2 || version.Patch() != 3 || version.Value() != raw {
		t.Errorf("version '%q' parsed incorrectly, got '%+v'", raw, version)
	}
}

func TestPipVersion_Error(t *testing.T) {
	version, err := NewPipVersion("hi1.2.3")
	if err == nil {
		t.Error("expected error on invalid version, got none")
	}
	if version != nil {
		t.Errorf("expected nil version on error, got '%+v'", version)
	}
}

func TestPipConstraints_Parts(t *testing.T) {
	raw := ">=1.2.3,<=1.4.0,  !=1.2.17"
	constr, err := NewPipConstraints(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if constr.Value() != raw {
		t.Fatalf("unexpected constraint value, expected '%q', got %q", raw, constr.Value())
	}
}

func TestPipConstraints_Error(t *testing.T) {
	constr, err := NewPipConstraints(">=1.2.3||<=1.4.0")
	if err == nil {
		t.Error("expected error on invalid constraint, got none")
	}
	if constr != nil {
		t.Errorf("expected nil version on error, got '%+v'", constr)
	}
}

func TestPipConstraintsAndVersion_MatchMethod(t *testing.T) {
	// Table test
	cases := []struct {
		Constraint string
		Version    string
		Result     bool
	}{
		{">=1.2.3,<=v1.4.0", "1.2.3", true},
		{">=1.2.3,<=v1.4.0", "1.3", true},
		{">=1.2.3,<=v1.4.0", "1.2.2", false},
		{">=1.2.3,<=v1.4.0", "1.4.0", true},
		{">=1.2.3,<=v1.4.0", "1.4.1", false},
		// Equals, wildcards
		{"3.*", "3.0", true},
		{"==3.*", "3.17.0", true},
		{"3.*", "3", true},
		{"3.7", "3.7", true},
		{"== 3.7", "3.7.0", true},
		{"3", "3.7.0", true},
		{"3", "3.7", true},
		{"3", "3", true},
		{"*", "3", true},
		{"==v3", "3.7.0", true},
		{"===v3", "3.7.0", false},
		{"===v3", "v3", true},
		// Not equals
		{"!=3.7", "3.7", false},
		{"!=3.7", "3.7.0", false},
		{"!=3.7,3.7", "3.7.0", false},
		{"!=3.7.*", "3.7.2", false},
		{"!=3.7.*", "3.8.2", true},
		{"!=3.*", "3.8.2", false},
		{"!=3.*", "4.8.2", true},
		// Simple comparison tests (>,<)
		{"<3.7", "3.6.0", true},
		{"<3.7.5", "3.7.4", true},
		{"<3.7.5", "3.7.5", false},
		{"<3.7.5", "3.7.6", false},
		{">3.7", "4.6.0", true},
		{"<3.7", "4.6.0", false},
		{">3.7", "2.6.0", false},
		{">3.7", "3.7.0", false},
		{">3.7", "3.7.1", true},
		{">3.7", "3.8.0", true},
		{">3.7.5", "3.7.6", true},
		{">3.7.5", "3.7.5", false},
		{">3.7.5", "3.7.4", false},
		// Tilda tests (~)
		{"~= 2.2", "2.3", true},
		{"~= 2.2", "2.1", false},
		{"~= 2.2", "3.0", false},
		{"~= 1.4.5", "1.4.1", false},
		{"~=1.2", "1.2", true},
		{"~=1.2", "1.2.0", true},
		{"~=1.2", "1.2.1", true},
		{"~=1.2", "1.8.99", true},
		{"~=1.2", "1.1", false},
		{"~=1.2", "2.0.0", false},
		{"~=1.2", "2.1.0", false},
		{"~=1.2.3", "1.2.3", true},
		{"~=1.2.3", "1.2.199", true},
		{"~=1.2.3", "2.0.0", false},
		{"~=0.0.0", "123.213.213", true},
		{"~=0.0.0", "0.0.0", true},
		{"~=*", "123.213.213", true},
	}

	for _, tcase := range cases {
		caseName := fmt.Sprintf("%q->%q)", tcase.Version, tcase.Constraint)
		t.Run(caseName, func(t *testing.T) {
			raw := tcase.Constraint
			constr, err := NewPipConstraints(raw)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if constr.Value() != raw {
				t.Fatalf("unexpected constraint value, expected '%q', got %q", raw, constr.Value())
			}

			ver, err := NewPipVersion(tcase.Version) // correct
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
