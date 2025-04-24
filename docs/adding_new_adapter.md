# 📖 Writing a New Input Adapter for sbommv

## Understanding Adapters in sbommv

`sbommv` uses a modular, pluggable architecture where adapters act as bridges between SBOM (Software Bill of Materials) sources and destinations. This design allows `sbommv` to support various input sources (e.g., S3 buckets, GitHub repositories, local folders) and output destinations (e.g., DependencyTrack, S3 buckets, local folders) without modifying the core logic.

- **Input Adapters** → Fetch SBOMs from source(e.g., from GitHub, Folder, etc.).
- **Output Adapters** → Send SBOMs to a destination(e.g., to Interlynk, Folder, etc.).
- **Adapter Interface**: All adapters implement the Adapter interface defined in pkg/adapter/factory.go, ensuring a consistent API for fetching, uploading, and dry-running SBOM transfers.

Each adapter is responsible for:

- Defining CLI flags for configuration.
- Validating user inputs.
- Fetching or uploading SBOMs.
- Supporting dry-run mode to simulate operations.

This guide focuses on adding a new input adapter, using the S3 input adapter (pkg/source/s3) as an example, but the process is similar for output adapters.

## Implementing a New Adapter

To add a new adapter, follow these steps. We’ll use the S3 input adapter as a reference, located in `pkg/source/s3`.

### Step 1: Create a Directory for the Adapter

- Create a new directory under `pkg/source/` for your adapter (e.g., `pkg/source/myadapter`).
- Example: The S3 input adapter lives in `pkg/source/s3/`.
- Typical files:
  - `adapter.go`: Defines the adapter struct and implements the Adapter interface.
  - `config.go`: Defines the configuration struct and methods (e.g., client initialization).
  - `fetcher.go`: Implements fetching logic (sequential and parallel modes).
  - `iterator.go`: Defines an iterator for lazy SBOM loading.
  - `reporter.go`: Handles dry-run reporting.

### Step 2: Define the Adapter Struct

Create a struct to hold the adapter’s configuration, role, processing mode, and fetcher/uploader. The struct must implement the **Adapter** interface.

**Example (S3 Adapter)**: In `pkg/source/s3/adapter.go`:

```go
type S3Adapter struct {
    Config         *S3Config
    Role           types.AdapterRole
    ProcessingMode types.ProcessingMode
    Fetcher        SBOMFetcher
}
```

Where:

- **Config**: holds adapter-specific settings (e.g., bucket name, region).
- **Role**: whether the adapter is for input (types.InputAdapterRole) or output (types.OutputAdapterRole).
- **ProcessingMode**: Specifies sequential or parallel processing (e.g., types.FetchSequential, types.FetchParallel).
- **Fetcher**: An interface for fetching SBOMs (e.g., SBOMFetcher for input adapters).

### Step 3: Implement the Adapter Interface

The **Adapter** interface in `pkg/adapter/factory.go` requires five methods:

```go
type Adapter interface {
    AddCommandParams(cmd *cobra.Command)
    ParseAndValidateParams(cmd *cobra.Command) error
    FetchSBOMs(ctx tcontext.TransferMetadata) (iterator.SBOMIterator, error)
    UploadSBOMs(ctx tcontext.TransferMetadata, iterator iterator.SBOMIterator) error
    DryRun(ctx tcontext.TransferMetadata, iterator iterator.SBOMIterator) error
}
```

- For an input adapter, you’ll implement all methods, but `UploadSBOMs` typically returns an error indicating it’s not supported (since input adapters fetch, not upload).

Where:

- **AddCommandParams** method: defines CLI flags specific to your adapter
- **ParseAndValidateParams** method: parses CLI flags and validates their values, populating the adapter’s configuration.
- **FetchSBOMs** method: fetches SBOMs from the source and returns an `iterator.SBOMIterator` for lazy processing.
- **UploadSBOMs** method: returns an error indicating that uploading is not supported for input role adapter.
- **DryRun** method: simulates fetching SBOMs, without performing actual operations.

### Step 5: Implement Fetcher Logic

- Define a fetcher interface and implementations for sequential and parallel fetching.
- **S3 Example** (`pkg/source/s3/fetcher.go`):

```go
type SBOMFetcher interface {
    Fetch(ctx tcontext.TransferMetadata, config *S3Config) (iterator.SBOMIterator, error)
}

type S3SequentialFetcher struct{}
type S3ParallelFetcher struct{}
```

- **Sequential**: Fetches SBOMs one-by-one.
- **Parallel**: Uses goroutines with a semaphore (e.g., maxConcurrency = 3) and mutex.

### Step 6: Implement Iterator

- Create an iterator to lazily yield SBOMs.
- **S3 Example** (`pkg/source/s3/iterator.go`):

```go
type S3Iterator struct {
    sboms []*iterator.SBOM
    index int
}

func NewS3Iterator(sboms []*iterator.SBOM) *S3Iterator {
    return &S3Iterator{
        sboms: sboms,
        index: 0,
    }
}

func (it *S3Iterator) Next(ctx tcontext.TransferMetadata) (*iterator.SBOM, error) {
    if it.index >= len(it.sboms) {
        return nil, io.EOF
    }
    sbom := it.sboms[it.index]
    it.index++
    return sbom, nil
}
```

### Step 7: Register the Adapter

- Add your adapter to the factory in `pkg/adapter/factory.go`.
- **S3 Example**:

```go
case types.S3AdapterType:
    adapters[types.InputAdapterRole] = &is3.S3Adapter{Role: types.InputAdapterRole, ProcessingMode: processingMode}
    inputAdp = "s3"
```

### Step 8: Let Transfer command know about this newly added adapter

- Under `registerAdapterFlags` function, create your adapter instance, and invoke `AddCommandParams` to sync the flags with `transfer` command.
- **S3 Example**:

```go
s3InputAdapter := &is3.S3Adapter{}
s3InputAdapter.AddCommandParams(cmd)
```

and lastly, under `parseConfig` function, allow the support for newly adapter. **S3Example**:

```go
validInputAdapter := map[string]bool{"github": true, "folder": true, "s3": true}
```

✅ Summary

By implementing the **Adapter** interface’s five methods (`AddCommandParams`, `ParseAndValidateParams`, `FetchSBOMs`, `UploadSBOMs`, `DryRun`) and following the S3 adapter’s structure, you can add a new input adapter to `sbommv`. The modular design ensures your adapter integrates seamlessly, allowing `sbommv` to fetch SBOMs from new sources without changing core logic.
