# dephub-core
Core Go libraries package used in DepHub project. Provide logic for managing (read-only, for now) dependencies for PHP and Python package managers.

## Internal packages

### API wrappers:
Helps you to communicate with different packages repositories.

#### [Packagist.org](https://packagist.org) wrapper ([package README.md](/providers/api/packagist/README.md))

Basic usage:
```go
// import "github.com/dephub/dephub-core/providers/api/packagist"

// Create new packagist api client, you can pass your httpClient.
p, err := packagist.NewClient(nil, nil)
if err != nil {
    panic(err)
}

// Options are used when there are optional parameters,
// you can pass nil if you dont need any.
options := &packagist.SearchOptions{
    Page: 3, // Optional page parameter for packagist search request
}

// Search query is 'laravel', context is usefull for cancelation,
// provide context.Background() if you dont need it.
results, response, err := p.Search(context.Background(), "laravel", options)
if err != nil {
    panic(err)
}

// You can use actual packagist response as you wish, usually you want to just omit it with _
fmt.Printf("Called %q url, got %d search results!\n", response.Request.URL, results.Total)

// output: Called "https://packagist.org/search.json?page=3&q=laravel" url, got 47071 search results!
```

##### [PyPi.org](https://pypi.org) wrapper ([package README.md](/providers/api/pip/README.md))

```go
// import "github.com/dephub/dephub-core/providers/api/pip"

// Create new PIP api client, you can pass your httpClient.
pip := pip.NewPyPiClient(http.DefaultClient, nil)

// Get all releases for Django 3.0.11
pkg, response, err := pip.Release(context.Background(), "Django", "3.0.11")
if err != nil {
	panic(err)
}

// You can use actual PyPi response as you wish, usually you want to just omit it with _
fmt.Printf("Called %q url, Django author: %q!\n", response.Request.URL, pkg.Info.Author)

// output: Called "https://pypi.org/pypi/Django/3.0.11/json" url, Django author: "Django Software Foundation"!
```

### Dependency files parsers ([package README.md](/providers/parsers/README.md)):
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

TODO: Documentation
