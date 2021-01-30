package dephub

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/dephub/dephub-core/providers/api/packagist"
	"github.com/dephub/dephub-core/providers/api/pip"
	"github.com/dephub/dephub-core/providers/versioneer"
)

// UpdatesChecker represents checkers interface.
type UpdatesChecker interface {
	// CompatibleUpdates returns latest available updates for locked dependencies compatible with constraints.
	CompatibleUpdates(ctx context.Context, constraints []Constraint, requirements []Requirement) ([]Update, error)
	// LastUpdates returns last releases for specified packages.
	LastUpdates(ctx context.Context, packages []Constraint, incompatibleOnly bool) ([]Update, error)
}

// Update represents one package update.
type Update struct {
	Version           string `json:"version"`
	Name              string `json:"name"`
	Author            string `json:"author"`
	URL               string `json:"url"`
	CurrentVersion    string `json:"current_version,omitempty"`
	CurrentConstraint string `json:"constraint,omitempty"`
}

// NewPIPUpdatesChecker constructs new PIPUpdatesChecker.
func NewPIPUpdatesChecker(httpClient *http.Client) UpdatesChecker {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	api := pip.NewPyPiClient(httpClient, nil)

	return &PIPUpdatesChecker{api: api}
}

// PIPUpdatesChecker represents PIP packages update checker.
type PIPUpdatesChecker struct {
	api pip.Client
}

// CompatibleUpdates returns latest available updates for locked dependencies compatible with constraints.
//
// Basically it is 'your locked dependency is lower then available with your constraints'
func (uc PIPUpdatesChecker) CompatibleUpdates(ctx context.Context, constraints []Constraint, requirements []Requirement) ([]Update, error) {
	if len(requirements) == 0 || len(constraints) == 0 {
		return nil, fmt.Errorf("no packages provided")
	}

	return []Update{}, nil
}

// Returns latest versions for each package
func (uc PIPUpdatesChecker) LastUpdates(ctx context.Context, packages []Constraint, incompatibleOnly bool) ([]Update, error) {
	if len(packages) == 0 {
		return nil, fmt.Errorf("no packages provided")
	}

	result := make([]Update, 0, len(packages))

skip_pkg:
	for _, pkg := range packages {
		update := &Update{}

		meta, _, err := uc.api.Release(ctx, pkg.Name, "")
		if err != nil {
			continue
		}

		constraint, err := versioneer.NewPipConstraints(pkg.Version)
		if err != nil {
			continue
		}

		for i := len(meta.Releases) - 1; i >= 0; i-- {
			vers, err := versioneer.NewPipVersion(meta.Releases[i].Version)
			if err != nil {
				continue
			}

			update.Name = pkg.Name
			update.Version = meta.Releases[i].Version
			update.Author = meta.Info.Author
			update.URL = meta.Info.ReleaseURL
			update.CurrentConstraint = pkg.Version

			// If we only need incompatible versions and the last version matches the constraint
			// then skip the package, it is already up do date
			if incompatibleOnly && constraint.Match(vers) {
				continue skip_pkg
			}

			break
		}

		if update != nil {
			result = append(result, *update)
		}
	}

	return result, nil
}

// NewComposerUpdatesChecker constructs new ComposerUpdatesChecker.
func NewComposerUpdatesChecker(httpClient *http.Client) UpdatesChecker {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	api, err := packagist.NewClient(httpClient, nil)
	if err != nil {
		panic(err)
	}

	return &ComposerUpdatesChecker{api: api}
}

// ComposerUpdatesChecker represents Composer packages update checker.
type ComposerUpdatesChecker struct {
	api packagist.Client
}

// CompatibleUpdates returns latest available updates for locked dependencies compatible with constraints.
//
// Basically it is 'your locked dependency is lower then available with your constraints'
func (uc ComposerUpdatesChecker) CompatibleUpdates(ctx context.Context, constraints []Constraint, requirements []Requirement) ([]Update, error) {
	if len(requirements) == 0 || len(constraints) == 0 {
		return nil, fmt.Errorf("no packages provided")
	}

	// To optimize requirements filtering
	reqsLookup := make(map[string]*Requirement)
	for i, req := range requirements {
		reqsLookup[req.Name] = &requirements[i]
	}

	result := make([]Update, 0, len(constraints))

	for _, cns := range constraints {
		if _, ok := reqsLookup[cns.Name]; !ok {
			continue
		}
		req := reqsLookup[cns.Name]

		metaData, err := uc.getPackagistMeta(ctx, uc.api, cns.Name)
		if err != nil {
			continue
		}

		update, err := uc.compatibleReleases(cns, *req, true, true, metaData)
		if err != nil {
			continue
		}

		if len(update) != 0 {
			result = append(result, *update[0])
		}
	}

	return result, nil
}

// compatibleReleases returns list of updates available for package
// updatable - show only constraint satisfying next versions
func (uc ComposerUpdatesChecker) compatibleReleases(constraint Constraint, req Requirement, updatable bool, first bool, meta packagist.PackageMeta) ([]*Update, error) {
	if len(meta) == 0 {
		return nil, fmt.Errorf("meta info is empty")
	}

	baseCst, err := versioneer.NewComposerConstraints(constraint.Version)
	if err != nil {
		return nil, err
	}

	reqCst, err := versioneer.NewComposerConstraints(">" + req.Version)
	if err != nil {
		return nil, err
	}

	releases := make([]*Update, 0, 5)

	// Filter parsable versions
	for i := len(meta) - 1; i >= 0; i-- {
		vers, err := versioneer.NewComposerVersion(meta[i].Version)
		if err != nil {
			continue
		}

		include := reqCst.Match(vers)
		if updatable {
			include = include && baseCst.Match(vers)
		}
		if include {
			update := composerVersionToUpdate(meta[i])
			update.CurrentVersion = req.Version
			update.CurrentConstraint = constraint.Version
			releases = append(releases, update)
			if first {
				return releases, nil
			}
		}
	}

	return releases, nil
}

// LastUpdates returns latest versions for each package
func (uc ComposerUpdatesChecker) LastUpdates(ctx context.Context, packages []Constraint, incompatibleOnly bool) ([]Update, error) {
	if len(packages) == 0 {
		return nil, fmt.Errorf("no packages provided")
	}

	result := make([]Update, 0, len(packages))

skip_pkg:
	for _, pkg := range packages {
		metaData, err := uc.getPackagistMeta(ctx, uc.api, pkg.Name)
		if err != nil {
			continue
		}

		constraint, err := versioneer.NewComposerConstraints(pkg.Version)
		if err != nil {
			continue
		}

		var update *Update
		// Filter first (from the newest) parsable version
		for i := len(metaData) - 1; i >= 0; i-- {
			vers, err := versioneer.NewComposerVersion(metaData[i].Version)
			if err != nil {
				continue
			}

			// If we only need incompatible versions and the last version matches the constraint
			// then skip the package, it is already up do date
			if incompatibleOnly && constraint.Match(vers) {
				continue skip_pkg
			}

			update = composerVersionToUpdate(metaData[i])
			break
		}

		if update != nil {
			update.CurrentConstraint = pkg.Version
			result = append(result, *update)
		}
	}

	return result, nil
}

// getPackagistMeta returns meta information about the package from packagist api.
func (uc ComposerUpdatesChecker) getPackagistMeta(ctx context.Context, cl packagist.Client, pkg string) (packagist.PackageMeta, error) {
	pkgNamePrts := strings.Split(pkg, "/")
	if len(pkgNamePrts) != 2 {
		return nil, fmt.Errorf("cannot parse vendor from package name %q", pkg)
	}
	vendor, name := pkgNamePrts[0], pkgNamePrts[1]
	metaData, _, err := cl.Meta(ctx, vendor, name)
	if err != nil {
		return nil, err
	}

	if _, ok := metaData.Packages[pkg]; !ok {
		return nil, fmt.Errorf("package %q not found on packagist", pkg)
	}
	return metaData.Packages[pkg], err
}

// composerVersionToUpdate is a little helper to convert VersionMeta to Update type.
func composerVersionToUpdate(release packagist.VersionMeta) *Update {
	update := &Update{
		Name:    release.Name,
		URL:     release.Source.URL,
		Version: release.Version,
		Author:  release.Name,
	}

	if len(release.Authors) != 0 {
		update.Author = release.Authors[0].Name
	}
	return update
}
