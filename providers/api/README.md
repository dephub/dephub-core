# API wrappers:

Helps you to communicate with different packages repositories.

#### [Packagist.org](https://packagist.org) wrapper

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

##### [PyPi.org](https://pypi.org) wrapper

Basic usage:

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
