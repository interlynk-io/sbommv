# 📖 Writing a New Adapter for sbommv

## Understanding Adapters in sbommv

`sbommv` follows a pluggable architecture where adapters act as interfaces between SBOM sources and destinations.

- **Input Adapters** → Fetch SBOMs (e.g., from GitHub, Folder, etc.).
- **Output Adapters** → Send SBOMs (e.g., to Interlynk, Folder, etc.).
- **Each Adapter Implements the** `Adapter` **Interface** → This ensures a common API across different sources & destinations.

## Implementing a New Adapter

To add a new adapter, follow these steps:

### Step 1: Define a Struct for Your Adapter

Each adapter has its own struct. This struct will hold relevant configuration details.

For **FolderAdapter**, we define:

```go
// FolderAdapter struct represents an adapter for local folder storage
type FolderAdapter struct {
	Role       types.AdapterRole
	FolderPath string
	Recursive  bool
}
```

💡 **Note**:

- **Role**: Defines whether the adapter is for input or output.
- **FolderPath**: Directory to scan or store SBOMs.
- **Recursive**: If true, scans subdirectories when acting as an input adapter.

### Step 2: Implement `AddCommandParams`

- This method adds CLI flags related to the adapter.

```go
// AddCommandParams adds folder adapter-specific CLI flags
func (f *FolderAdapter) AddCommandParams(cmd *cobra.Command) {
// implementation code
}
```

- This ensures the correct flags are registered depending on the adapter role.

### Step 3: Implement `ParseAndValidateParams`

- This method validates and extracts CLI parameters.

```go
// ParseAndValidateParams extracts and validates folder adapter parameters
func (f *FolderAdapter) ParseAndValidateParams(cmd *cobra.Command) error {
// implementation code
}
```

- This ensures we have valid folder paths before proceeding.

### Step 4: Implement `FetchSBOMs` for Input Adapter

- This method scans a folder, detects SBOMs, and returns an iterator.

```go
// FetchSBOMs retrieves SBOMs from the specified folder
func (f *FolderAdapter) FetchSBOMs(ctx *tcontext.TransferMetadata) (iterator.SBOMIterator, error) {
// implementation code
}
```

- This method:

  - Scans the directory recursively, if resursive flag is `true`.
  - Detects SBOMs using utils.IsValidSBOM().
  - Returns an iterator for processing SBOMs.

### Step 5: Implement UploadSBOMs for Output Adapter

- This method saves SBOMs to the specified folder.

```go
// UploadSBOMs writes SBOMs to the specified folder
func (f *FolderAdapter) UploadSBOMs(ctx *tcontext.TransferMetadata, it iterator.SBOMIterator) error {
  // implementation code
}
```

- This method:
  - Creates a folder if it doesn’t exist.
  - Writes SBOMs using either their original filename or a generated UUID.

### Step 6: Implement DryRun

```go
// DryRun simulates fetching or uploading SBOMs
func (f *FolderAdapter) DryRun(ctx *tcontext.TransferMetadata, it iterator.SBOMIterator) error {
// implementation code
}
```

- This method:

  - In input mode, lists detected SBOMs.
  - In output mode, shows where SBOMs will be saved.
