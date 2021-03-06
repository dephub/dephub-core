package dephub

import (
	"context"
	"net/http"
	"testing"

	"github.com/dephub/dephub-core/providers/api/packagist"
	"github.com/dephub/dephub-core/providers/api/pip"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type PackagistMock struct {
	mock.Mock
	packagist.PackagistClient
}

func (mock *PackagistMock) Meta(ctx context.Context, vendor, pkg string) (*packagist.PackagesMeta, *http.Response, error) {
	args := mock.Called(ctx, vendor, pkg)
	var f *packagist.PackagesMeta
	var s *http.Response
	// To allow nil values
	if mt, ok := args.Get(0).(*packagist.PackagesMeta); ok {
		f = mt
	}
	if resp, ok := args.Get(1).(*http.Response); ok {
		s = resp
	}

	return f, s, args.Error(2)
}

// PyPiMock mocks PyPiClient logic.
type PyPiMock struct {
	mock.Mock
	pip.PyPiClient
}

// Mock Release method.
func (mock *PyPiMock) Release(ctx context.Context, name, version string) (*pip.PipPackage, *http.Response, error) {
	args := mock.Called(ctx, name, version)
	var f *pip.PipPackage
	var s *http.Response
	// To allow nil values
	if mt, ok := args.Get(0).(*pip.PipPackage); ok {
		f = mt
	}
	if resp, ok := args.Get(1).(*http.Response); ok {
		s = resp
	}

	return f, s, args.Error(2)
}

func TestComposerUpdatesChecker_NewMethod(t *testing.T) {
	cl := NewComposerUpdatesChecker(nil)
	assert.True(t, cl.(*ComposerUpdatesChecker).api != nil)
}

func TestComposerUpdatesChecker_LastUpdatesMethod(t *testing.T) {
	coreSource := NewMemorySource(sourceMockFileStorage)
	// Set our mock to always return one result on every Meta call.
	apiMock := new(PackagistMock)
	apiMock.On("Meta", mock.Anything, mock.Anything, mock.Anything).Return(&composerPackagesMeta, nil, nil)

	expectedUpdates := []Update{
		{Name: "testing/something", Author: "testing/something", Version: "2.1.17", CurrentConstraint: "2.0.*"},
		{Name: "another/testpackage", Author: "another/testpackage", Version: "3.7.0", CurrentConstraint: "3.5.*"},
	}

	uc := ComposerUpdatesChecker{api: apiMock}

	constraints, err := coreSource.Constraints(context.Background(), ComposerType)
	if err != nil {
		t.Fatalf("unexpected error on source constraints: %v", err)
	}

	updates, err := uc.LastUpdates(context.Background(), constraints, true)
	if err != nil {
		t.Fatalf("unexpected error on last updates: %v", err)
	}

	assert.Len(t, updates, 2)
	assert.ElementsMatch(t, expectedUpdates, updates)
	apiMock.AssertExpectations(t)
}

func TestComposerUpdatesChecker_LastUpdatesMethod_WithCompatible(t *testing.T) {
	coreSource := NewMemorySource(sourceMockFileStorage)
	// Set our mock to always return one result on every Meta call.
	apiMock := new(PackagistMock)
	apiMock.On("Meta", mock.Anything, mock.Anything, mock.Anything).Return(&composerPackagesMeta, nil, nil)

	expectedUpdates := []Update{
		{Name: "testing/something", Author: "testing/something", Version: "2.1.17", CurrentConstraint: "2.0.*"},
		{Name: "another/testpackage", Author: "another/testpackage", Version: "3.7.0", CurrentConstraint: "3.5.*"},
		{Name: "test/package", Author: "test/package", Version: "1.2.3", CurrentConstraint: ">=1.0.0"},
	}

	uc := ComposerUpdatesChecker{api: apiMock}

	constraints, err := coreSource.Constraints(context.Background(), ComposerType)
	if err != nil {
		t.Fatalf("unexpected error on source constraints: %v", err)
	}

	updates, err := uc.LastUpdates(context.Background(), constraints, false)
	if err != nil {
		t.Fatalf("unexpected error on last updates: %v", err)
	}

	assert.Len(t, updates, 3)
	assert.ElementsMatch(t, expectedUpdates, updates)
	apiMock.AssertExpectations(t)
}

func TestComposerUpdatesChecker_CompatibleUpdatesMethod(t *testing.T) {
	coreSource := NewMemorySource(sourceMockFileStorage)
	// Set our mock to always return one result on every Meta call.
	apiMock := new(PackagistMock)
	apiMock.On("Meta", mock.Anything, mock.Anything, mock.Anything).Return(&composerPackagesMeta, nil, nil)

	expectedUpdates := []Update{
		{Name: "another/testpackage", Author: "another/testpackage", Version: "3.5.19", CurrentVersion: "v3.5.2", CurrentConstraint: "3.5.*"},
		{Name: "testing/something", Author: "testing/something", Version: "2.0.3", CurrentVersion: "v1.9.17", CurrentConstraint: "2.0.*"},
	}

	uc := ComposerUpdatesChecker{api: apiMock}

	constraints, err := coreSource.Constraints(context.Background(), ComposerType)
	if err != nil {
		t.Fatalf("unexpected error on source constraints: %v", err)
	}
	reqs, err := coreSource.Requirements(context.Background(), ComposerType)
	if err != nil {
		t.Fatalf("unexpected error on source constraints: %v", err)
	}

	updates, err := uc.CompatibleUpdates(context.Background(), []Constraint{}, []Requirement{})
	if err == nil || err.Error() != "no packages provided" {
		t.Error("expected error on empty packages, got none")
	}
	assert.Len(t, updates, 0)

	updates, err = uc.CompatibleUpdates(context.Background(), constraints, reqs)
	if err != nil {
		t.Error("expected no errors, got: %w", err)
	}

	assert.Len(t, updates, 2)
	assert.ElementsMatch(t, expectedUpdates, updates)
	apiMock.AssertExpectations(t)
}

func TestPIPUpdatesChecker_NewMethod(t *testing.T) {
	cl := NewPIPUpdatesChecker(nil)
	assert.True(t, cl.(*PIPUpdatesChecker).api != nil)
}

func TestPIPUpdatesChecker_LastUpdatesMethod(t *testing.T) {
	coreSource := NewMemorySource(sourceMockFileStorage)

	// Set our mock to always return one result on every Meta call.
	apiMock := new(PyPiMock)
	apiMock.On("Release", mock.Anything, "MyPackage", mock.Anything).Return(pipReleases["MyPackage"], nil, nil)
	apiMock.On("Release", mock.Anything, "AnotherPackage", mock.Anything).Return(pipReleases["AnotherPackage"], nil, nil)
	apiMock.On("Release", mock.Anything, "testing-test", mock.Anything).Return(pipReleases["testing-test"], nil, nil)

	expectedUpdates := []Update{
		{Name: "AnotherPackage", Author: "another package author", Version: "1.3", CurrentConstraint: "==1.1.0"},
		{Name: "testing-test", Author: "testing-test package author", Version: "3.17.6", CurrentConstraint: ">=2.4.2,<3.17.6"},
	}

	uc := PIPUpdatesChecker{api: apiMock}

	constraints, err := coreSource.Constraints(context.Background(), PIPType)
	if err != nil {
		t.Fatalf("unexpected error on source constraints: %v", err)
	}

	updates, err := uc.LastUpdates(context.Background(), constraints, true)
	if err != nil {
		t.Fatalf("unexpected error on last updates: %v", err)
	}

	assert.Len(t, updates, 2)
	assert.ElementsMatch(t, expectedUpdates, updates)
	apiMock.AssertExpectations(t)
}

func TestPIPUpdatesChecker_LastUpdatesMethod_WithCompatible(t *testing.T) {
	coreSource := NewMemorySource(sourceMockFileStorage)

	// Set our mock to always return one result on every Meta call.
	apiMock := new(PyPiMock)
	apiMock.On("Release", mock.Anything, "MyPackage", mock.Anything).Return(pipReleases["MyPackage"], nil, nil)
	apiMock.On("Release", mock.Anything, "AnotherPackage", mock.Anything).Return(pipReleases["AnotherPackage"], nil, nil)
	apiMock.On("Release", mock.Anything, "testing-test", mock.Anything).Return(pipReleases["testing-test"], nil, nil)

	expectedUpdates := []Update{
		{Name: "MyPackage", Author: "my package author", Version: "3.1.4", CurrentConstraint: "==3.1.4"},
		{Name: "AnotherPackage", Author: "another package author", Version: "1.3", CurrentConstraint: "==1.1.0"},
		{Name: "testing-test", Author: "testing-test package author", Version: "3.17.6", CurrentConstraint: ">=2.4.2,<3.17.6"},
	}

	uc := PIPUpdatesChecker{api: apiMock}

	constraints, err := coreSource.Constraints(context.Background(), PIPType)
	if err != nil {
		t.Fatalf("unexpected error on source constraints: %v", err)
	}

	updates, err := uc.LastUpdates(context.Background(), constraints, false)
	if err != nil {
		t.Fatalf("unexpected error on last updates: %v", err)
	}

	assert.Len(t, updates, 3)
	assert.ElementsMatch(t, expectedUpdates, updates)
	apiMock.AssertExpectations(t)
}

func TestPIPUpdatesChecker_CompatibleUpdatesMethod(t *testing.T) {
	// Set our mock to always return one result on every Meta call.
	apiMock := new(PyPiMock)
	apiMock.AssertNotCalled(t, "Release", mock.Anything, mock.Anything, mock.Anything)

	uc := PIPUpdatesChecker{api: apiMock}

	updates, err := uc.CompatibleUpdates(context.Background(), []Constraint{}, []Requirement{})
	if err == nil || err.Error() != "no packages provided" {
		t.Error("expected error on empty packages, got none")
	}
	assert.Len(t, updates, 0)

	updates, err = uc.CompatibleUpdates(context.Background(), []Constraint{{}}, []Requirement{{}})
	if err != nil {
		t.Error("expected no errors, got: %w", err)
	}

	assert.Len(t, updates, 0)
	apiMock.AssertExpectations(t)
}

var composerPackagesMeta = packagist.PackagesMeta{Packages: map[string]packagist.PackageMeta{
	"test/package": {
		{Version: "0.8.19", Name: "test/package"},
		{Version: "1.1.7", Name: "test/package"},
		{Version: "1.2.3", Name: "test/package"},
	},
	"another/testpackage": {
		{Version: "3.2.5", Name: "another/testpackage"},
		{Version: "3.5.19", Name: "another/testpackage"},
		{Version: "3.7.0", Name: "another/testpackage"},
	},
	"testing/something": {
		{Version: "1.2.5", Name: "testing/something"},
		{Version: "1.5.19", Name: "testing/something"},
		{Version: "2.0.3", Name: "testing/something"},
		{Version: "2.1.17", Name: "testing/something"},
	},
}}

var pipReleases = map[string]*pip.PipPackage{
	"MyPackage": {
		Info: pip.PipPackageInfo{
			Author: "my package author",
			Name:   "MyPackage",
		},
		Releases: pip.PipPackageVersions{
			{Version: "1.7.2"},
			{Version: "2.2.0"},
			{Version: "3.1.4"},
		},
	},
	"AnotherPackage": {
		Info: pip.PipPackageInfo{
			Author: "another package author",
			Name:   "AnotherPackage",
		},
		Releases: pip.PipPackageVersions{
			{Version: "0.7.2"},
			{Version: "1.0.3"},
			{Version: "1.1.0"},
			{Version: "1.3"},
		},
	},
	"testing-test": {
		Info: pip.PipPackageInfo{
			Author: "testing-test package author",
			Name:   "testing-test",
		},
		Releases: pip.PipPackageVersions{
			{Version: "2.4.1"},
			{Version: "3.17.6"},
		},
	},
}

var sourceMockFileStorage = map[string][]byte{
	"composer.json": []byte(`
		{
			"require": {
				"php": ">=7.1.3",
				"test/package": ">=1.0.0",
				"another/testpackage": "3.5.*",
				"testing/something": "2.0.*"
			}
		}
	`),
	"composer.lock": []byte(`
		{
			"packages": [
				{
					"name": "test/package",
					"version": "1.2.4"
				},
				{
					"name": "another/testpackage",
					"version": "v3.5.2"
				},
				{
					"name": "testing/something",
					"version": "v1.9.17"
				}
			]
		}
	`),
	"requirements.txt": []byte(`
			MyPackage==3.1.4
			AnotherPackage==1.1.0
			testing-test>=2.4.2,<3.17.6
	`),
}
