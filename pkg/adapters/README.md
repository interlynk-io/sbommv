# SBOM Collection Package

This package provides a unified interface for collecting Software Bill of Materials (SBOM) documents from various sources including files, folders, GitHub repositories, AWS S3 buckets, and the Interlynk platform.

## Features

- Multiple source adapters:
  - File/Directory: Read SBOMs from local files or scan directories
  - GitHub: Retrieve SBOMs from releases, API, or generate them
  - AWS S3: Scan S3 buckets for SBOM files
  - Interlynk: Integration with the Interlynk platform

- Supported SBOM formats:
  - CycloneDX
  - SPDX
  - Auto-format detection

## Installation

```bash
go get github.com/yourusername/sbom-collector
```

## Usage Examples

### Reading from Files

```go
// Read a single SBOM file
adapter, err := pkg.NewFileAdapter("path/to/sbom.json", pkg.InputOptions{})
if err != nil {
    log.Fatal(err)
}

sboms, err := adapter.GetSBOMs(context.Background())
if err != nil {
    log.Fatal(err)
}

// Read all SBOMs from a directory
adapter, err = pkg.NewFileAdapter("path/to/sboms/", pkg.InputOptions{
    IncludeFormats: []pkg.SBOMFormat{pkg.FormatCycloneDX},
})
```

### Using GitHub

```go
adapter := pkg.NewGitHubAdapter(
    "owner",
    "repo",
    "github-token",
    pkg.MethodReleases,
    pkg.InputOptions{},
)

sboms, err := adapter.GetSBOMs(context.Background())
```

### Using AWS S3

```go
adapter, err := pkg.NewS3Adapter(
    "my-bucket",
    "sboms/",
    pkg.InputOptions{
        MaxConcurrent: 5,
    },
)
if err != nil {
    log.Fatal(err)
}

sboms, err := adapter.GetSBOMs(context.Background())
```

### Using Interlynk

```go
adapter := pkg.NewInterlynkAdapter(
    "project-id",
    "https://api.interlynk.io",
    "api-key",
    pkg.InputOptions{},
)

sboms, err := adapter.GetSBOMs(context.Background())
```

## Input Options

The `InputOptions` struct allows you to configure how adapters operate:

```go
opts := pkg.InputOptions{
    // Maximum number of concurrent operations
    MaxConcurrent: 5,
    
    // Only include specific formats
    IncludeFormats: []pkg.SBOMFormat{
        pkg.FormatCycloneDX,
        pkg.FormatSPDX,
    },
    
    // Exclude specific formats
    ExcludeFormats: []pkg.SBOMFormat{
        pkg.FormatUnknown,
    },
}
```

## Error Handling

All adapters implement robust error handling and will return detailed error messages when operations fail:

```go
sboms, err := adapter.GetSBOMs(context.Background())
if err != nil {
    switch {
    case errors.Is(err, os.ErrNotExist):
        // Handle file not found
    case errors.Is(err, context.DeadlineExceeded):
        // Handle timeout
    default:
        // Handle other errors
    }
}
```
