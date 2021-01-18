package parsers

import (
	"context"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/dephub/dephub-core/providers/fetchers"
)

func TestComposerConstraintsMethod(t *testing.T) {
	bf := fetchers.ByteMapFetcher{Files: map[string][]byte{
		"composer.json": []byte(`{
			"name": "laravel/laravel",
			"description": "The Laravel Framework.",
			"require": {
				"php": ">=7.1.3",
				"fideloper/proxy": "^4.0",
				"laravel/framework": "5.7.*",
				"laravel/tinker": "~1.0"
			},
			"require-dev": {
				"filp/whoops": "~2.0",
				"fzaninotto/faker": "~1.4"
			}
		}`),
	}}
	parser := NewComposerParser(bf)

	cns, err := parser.Constraints(context.Background())
	if err != nil {
		t.Errorf("unexpected error on composer constraints call : %v", err)
	}

	expectedConstraints := []Constraint{
		{Name: "php", Version: ">=7.1.3"},
		{Name: "fideloper/proxy", Version: "^4.0"},
		{Name: "laravel/framework", Version: "5.7.*"},
		{Name: "laravel/tinker", Version: "~1.0"},
	}

	// Sort before DeepEqual test
	sort.Slice(cns, func(i, j int) bool {
		return cns[i].Name > cns[j].Name
	})
	sort.Slice(expectedConstraints, func(i, j int) bool {
		return expectedConstraints[i].Name > expectedConstraints[j].Name
	})

	if !reflect.DeepEqual(cns, expectedConstraints) {
		t.Errorf("unexpected composer constraints, got: '%+v", cns)
	}
}

func TestComposerConstraintsMethod_Errors(t *testing.T) {
	// Table test cases
	cases := []struct {
		Name  string
		Files map[string][]byte
		Err   string
	}{
		{"1", map[string][]byte{"blablabla": []byte("{}")}, ErrFileNotFound.Error()},
		{"1", map[string][]byte{"composer.json": []byte("broken")}, "unable to parse composer file content"},
	}

	for _, v := range cases {
		t.Run(v.Name, func(t *testing.T) {
			bf := fetchers.ByteMapFetcher{Files: v.Files}
			parser := NewComposerParser(bf)

			cns, err := parser.Constraints(context.Background())
			if err == nil || !strings.Contains(err.Error(), v.Err) {
				t.Error("expected error, got none")
			}
			if cns != nil {
				t.Errorf("expected nil constraints, got: %+v", cns)
			}
		})
	}
}

func TestComposerRequirementsMethod(t *testing.T) {
	bf := fetchers.ByteMapFetcher{Files: map[string][]byte{
		"composer.lock": []byte(`{
			"_readme": [
				"This file locks the dependencies of your project to a known state",
				"Read more about it at https://getcomposer.org/doc/01-basic-usage.md#installing-dependencies",
				"This file is @generated automatically"
			],
			"content-hash": "b4eeb50c248b397e208a7bd7d7f470b6",
			"packages": [
				{
					"name": "aws/aws-sdk-php",
					"version": "3.69.16"
				},
				{
					"name": "vlucas/phpdotenv",
            		"version": "v2.5.1"
				}
			]
		}`),
	}}
	parser := NewComposerParser(bf)

	reqs, err := parser.Requirements(context.Background())
	if err != nil {
		t.Errorf("unexpected error on composer constraints call : %v", err)
	}

	expectedRequirements := []Requirement{
		{Name: "aws/aws-sdk-php", Version: "3.69.16"},
		{Name: "vlucas/phpdotenv", Version: "v2.5.1"},
	}

	// Sort before DeepEqual test
	sort.Slice(reqs, func(i, j int) bool {
		return reqs[i].Name > reqs[j].Name
	})
	sort.Slice(expectedRequirements, func(i, j int) bool {
		return expectedRequirements[i].Name > expectedRequirements[j].Name
	})

	if !reflect.DeepEqual(reqs, expectedRequirements) {
		t.Errorf("unexpected composer constraints, got: '%+v", reqs)
	}
}

func TestComposerRequirementsMethod_Errors(t *testing.T) {
	// Table test cases
	cases := []struct {
		Name  string
		Files map[string][]byte
		Err   string
	}{
		{"1", map[string][]byte{"blablabla": []byte("{}")}, ErrFileNotFound.Error()},
		{"1", map[string][]byte{"composer.lock": []byte("broken")}, "unable to parse composer file content"},
	}

	for _, v := range cases {
		t.Run(v.Name, func(t *testing.T) {
			bf := fetchers.ByteMapFetcher{Files: v.Files}
			parser := NewComposerParser(bf)

			cns, err := parser.Requirements(context.Background())
			if err == nil || !strings.Contains(err.Error(), v.Err) {
				t.Error("expected error, got none")
			}
			if cns != nil {
				t.Errorf("expected nil constraints, got: %+v", cns)
			}
		})
	}
}
