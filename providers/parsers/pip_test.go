package parsers

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"testing"

	"github.com/dephub/dephub-core/providers/fetchers"
)

func TestPipRequirementsMethod(t *testing.T) {
	bf := fetchers.ByteMapFetcher{Files: map[string][]byte{
		"requirements.txt": []byte(requirementsTxtFixture),
	}}
	parser := NewPipParser(bf, "test")

	val, err := parser.Requirements(context.Background())
	if val != nil || err != nil {
		t.Errorf("expected nills on pip requirements call, got: '%+v', '%+v'", val, err)
	}
}

func TestPipParserConstraintsMethod(t *testing.T) {
	bf := fetchers.ByteMapFetcher{Files: map[string][]byte{
		"requirements.txt": []byte(requirementsTxtFixture),
	}}
	parser := NewPipParser(bf, "")

	reqs, err := parser.Constraints(context.Background())
	if err != nil {
		t.Errorf("unexpected error on pip constraints call : %v", err)
	}

	expectedRequirements := []Constraint{
		{Name: "coverage", Version: "!=3.5"},
		{Name: "rejected", Version: "*"},
		{Name: "nose-cov", Version: "*"},
		{Name: "docopt", Version: "==0.6.1"},
		{Name: "keyring", Version: ">=4.1.1"},
		{Name: "Mopidy-Dirble", Version: "~=1.1"},
		{Name: "green", Version: "*"},
		{Name: "hose", Version: "*"},
		{Name: "beautifulsoup4", Version: "*"},
	}

	// Sort before DeepEqual test
	sort.Slice(reqs, func(i, j int) bool {
		return reqs[i].Name > reqs[j].Name
	})
	sort.Slice(expectedRequirements, func(i, j int) bool {
		return expectedRequirements[i].Name > expectedRequirements[j].Name
	})

	if !reflect.DeepEqual(reqs, expectedRequirements) {
		fmt.Println(len(reqs), len(expectedRequirements))
		t.Errorf("unexpected pip constraints, got: '%+v", reqs)
	}
}

func TestPipParserConstraintsMethod_Errors(t *testing.T) {
	bf := fetchers.ByteMapFetcher{Files: map[string][]byte{
		"anotherfile.txt": []byte(requirementsTxtFixture),
	}}
	parser := NewPipParser(bf, "")

	reqs, err := parser.Constraints(context.Background())
	if err == nil {
		t.Error("expected error on missing pip files constraints call, got none")
	}
	if reqs != nil {
		t.Errorf("expected nil result on mission pip file, got '%+v'", reqs)
	}
}

var requirementsTxtFixture = `####### example-requirements.txt #######
#
###### Requirements without Version Specifiers ######
hose
nose-cov
beautifulsoup4
#
###### Requirements with Version Specifiers ######
#   See https://www.python.org/dev/peps/pep-0440/#version-specifiers
docopt == 0.6.1             # Version Matching. Must be version 0.6.1
keyring >= 4.1.1            # Minimum version 4.1.1
coverage != 3.5             # Version Exclusion. Anything except version 3.5
Mopidy-Dirble ~= 1.1        # Compatible release. Same as >= 1.1, == 1.*
#
###### Refer to other requirements files ######
-r other-requirements.txt
#

#
###### A particular file ######
./downloads/numpy-1.9.2-cp34-none-win32.whl
http://wxpython.org/Phoenix/snapshot-builds/wxPython_Phoenix-3.0.3.dev1820+49a8884-cp34-none-win_amd64.whl
#
###### Additional Requirements without Version Specifiers ######
#   Same as 1st section, just here to show that you can put things in any order.
rejected
green
#`
