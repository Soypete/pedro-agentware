# Installation

## Requirements

- Go 1.21 or later
- Access to Kitaru server (local or Kubernetes)

## Install SDK

```bash
go get github.com/zenml-io/kitaru-sdk-go@latest
```

Or specify a version:

```bash
go get github.com/zenml-io/kitaru-sdk-go@v0.13.1
```

## Verify Installation

Create a test file to verify the SDK installs correctly:

```go
package main

import (
    "fmt"
    kitaru "github.com/zenml-io/kitaru-sdk-go"
)

func main() {
    fmt.Printf("Kitaru SDK Version: %s\n", kitaru.Version)
}
```

Run it:

```bash
go run your-test-file.go
```

## Project Setup

Initialize a new Go module if you don't have one:

```bash
go mod init your-project-name
go mod tidy
```

## Dependencies

The SDK depends on:

- `github.com/google/uuid` - For execution ID handling
- `github.com/go-resty/resty/v2` - For HTTP client

These are automatically installed with the SDK.

## Platform Support

- Linux (amd64, arm64)
- macOS (amd64, arm64)
- Windows (amd64)

The SDK connects to Kitaru over HTTP, so the client can run anywhere with network access to the Kitaru server.