package dephub

import (
	"context"
	"fmt"
	"net/http"

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
