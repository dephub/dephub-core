# DepHub Core

Set of libraries, providing functionality for managing (read-only) dependencies for PHP and Python package managers.

> :exclamation: The package is in active developement. Methods may and will change over time until the first major release (1.\*). Then the project will follow semantic versioning rules.

## Packages

There are several useful packages included:

- API wrappers for fetching additional information on packages ([package README.md](/providers/api/README.md)):
  - Packagist API
  - PyPi API
- Source fetchers ([package README.md](/providers/fetchers/README.md))
- Dependency files parsers ([package README.md](/providers/parsers/README.md))
- Versions and constraints parser with checking logic (`/providers/versioneer`)

## Main `dephub` module

This module provides functionality for fetching, parsing and working with different package managers files and services.

### Fetching dependency managers files

Dependency source represents abstraction over package manager source files and provides convinient interface to fetch packages information.

Usage example (retrieving `Composer` constraints list):

```go
// import "github.com/dephub/dephub-core/dephub"

// We should define the source where our dep files are stored, in this example we will use the Git source.
// The last argument can be blank.
source, err := dephub.NewGitSource(http.DefaultClient, "git@github.com:laravel/framework.git", "master")
if err != nil {
    panic(err)
}

// Get all the constraints for Composer from the source.
constraints, err := source.Constraints(context.Background(), dephub.ComposerType)
if err != nil {
    panic(err)
}

fmt.Printf("Random constraint from \"laravel/framework\" composer.json: %q:%q \n", constraints[0].Name, constraints[0].Version)
// output: Random constraint from "laravel/framework" composer.json: "psr/simple-cache":"^1.0"
```

### Packages updates checking

Dependency checkers allow you to check constraints and requirements and get new/updatable versions information for them.

Usage example:

```go
source, _ := dephub.NewGitSource(http.DefaultClient, "git@github.com:laravel/framework.git", "master")
constraints, _ := source.Constraints(context.Background(), dephub.ComposerType)

updatesChecker := dephub.NewComposerUpdatesChecker(http.DefaultClient)
incompatibleOnly := true
updates, err := updatesChecker.LastUpdates(context.Background(), constraints, incompatibleOnly)
if err != nil {
    panic(err)
}

firstUpdate := updates[0]
fmt.Printf("Package %q (current constraint %q) has new version %q, get info on %q", firstUpdate.Name, firstUpdate.CurrentConstraint, firstUpdate.Version, firstUpdate.URL)
// output: Package "monolog/monolog" (current constraint "^2.0") has new version "2.2.0", get info on "https://github.com/Seldaek/monolog.git"
```
