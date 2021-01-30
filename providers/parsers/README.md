# Dependency files parsers:

Provide dependency files parsers (e.g. `composer.lock` or `requirements.txt`)

#### [Composer](https://getcomposer.org) dependency parser

Basic usage:

```go
// 	import "github.com/dephub/dephub-core/providers/fetchers"
// 	import "github.com/dephub/dephub-core/providers/parsers"

// Each parser requires a source from where they fetch dependency files.
fileFetcher := fetchers.NewGitHubFetcher(http.DefaultClient, "laravel", "framework", "master")
// Create new composer dependencies parser.
depParser := parsers.NewComposerParser(fileFetcher)
// Get constraints (composer.json require packages info),
// you can get composer.lock packages by calling Requirements method.
constraints, err := depParser.Constraints(context.Background())
if err != nil {
	panic(err)
}

// It is possible we could get zero constraints if composer.json has none
constraint := parsers.Constraint{Name: "NOT_FOUND"}
for _, cnst := range constraints {
	constraint = cnst
	break
}

fmt.Printf("Random composer.json package %q in 'laravel' repository has %q constraint\n", constraint.Name, constraint.Version)
// output: Random composer.json package "league/flysystem" in 'laravel' repository has "^2.0" constraint
```

#### [PIP](https://pypi.org/project/pip) dependency parser

Basic usage:

```go
// 	import "github.com/dephub/dephub-core/providers/fetchers"
// 	import "github.com/dephub/dephub-core/providers/parsers"

// Each parser requires a source from where they fetch dependency files.
fileFetcher := fetchers.NewGitHubFetcher(http.DefaultClient, "pallets", "flask", "master")

// Create new PIP dependencies parser, you can omit the filename, default is 'requirements.txt'.
depParser := parsers.NewPipParser(fileFetcher, "requirements/dev.txt")
// Parse constraints from dev flask file.
constraints, err := depParser.Constraints(context.Background())
if err != nil {
	panic(err)
}

// It is possible we could get zero constraints if file is empty
constraint := parsers.Constraint{Name: "NOT_FOUND"}
for _, cnst := range constraints {
	constraint = cnst
	break
}

fmt.Printf("Random PIP package %q in 'flask' repository has %q constraint\n", constraint.Name, constraint.Version)
// output: Random PIP package "toml" in 'flask' repository has "==0.10.2" constraint
```
