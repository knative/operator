# client-go-client

A [client-go](https://github.com/kubernetes/client-go) implementation
of the [Manifestival](https://github.com/manifestival/manifestival)
`Client`.

Usage
-----

```go
import (
    mfc "github.com/manifestival/client-go-client"
    mf  "github.com/manifestival/manifestival"
    "k8s.io/client-go/rest"
)

func main() {
    var config *rest.Config = ...
    
    manifest, err := mfc.NewManifest("file.yaml", config)
    if err != nil {
        panic("Failed to load manifest")
    }
    manifest.Apply()

    // a slightly more complex example
    m, err := mf.ManifestFrom(mf.Recursive("dir/"), mf.UseClient(mfc.NewClient(config)))
    if err != nil {
        panic("Failed to load manifest")
    }
    m.Apply()
}
```

The `NewManifest` function in this library delegates to the function
of the same name in the `manifestival` package after constructing a
`manifestival.Client` implementation from the `*rest.Config`.
