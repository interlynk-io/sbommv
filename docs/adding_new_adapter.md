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

This method defines adapter-specific CLI flags.

```go
func (f *FolderAdapter) AddCommandParams(cmd *cobra.Command) {
	// Register CLI flags (e.g., folder path, recursive, etc.)
}
```

This ensures users can configure the adapter via command-line arguments.

### Step 3: Implement `ParseAndValidateParams`

Parses and validates the CLI input passed to the adapter.

```go
func (f *FolderAdapter) ParseAndValidateParams(cmd *cobra.Command) error {
	// Extract and validate parameters like folder path
}
```

This ensures validating folder configuration values before proceeding.

### Step 4: Implement `FetchSBOMs` for Input Adapter

Responsible for retrieving SBOMs from the source.

```go
func (f *FolderAdapter) FetchSBOMs(ctx *tcontext.TransferMetadata) (iterator.SBOMIterator, error) {
	// Scan folder and return an iterator over SBOMs
}
```

This method:

- Scans the directory recursively, if resursive flag is `true`, otherwise only scans parent directory.
- Identify valid SBOMs using utilities like  `utils.IsValidSBOM()`.
- Returns an iterator for processing SBOMs.

### Step 5: Implement UploadSBOMs for Output Adapter

Responsible for uploading or storing SBOMs to the destination.

```go
// UploadSBOMs writes SBOMs to the specified folder
func (f *FolderAdapter) UploadSBOMs(ctx *tcontext.TransferMetadata, it iterator.SBOMIterator) error {
	// Write SBOMs to the target folder
}
```

This method:

- Create the output folder if it doesn’t exist.
- Write SBOMs using original or generated filenames via UUID.

### Step 6: Implement DryRun

Simulates the adapter’s behavior without real file transfers.

```go
func (f *FolderAdapter) DryRun(ctx *tcontext.TransferMetadata, it iterator.SBOMIterator) error {
	// Log what would be fetched (input) or saved (output)
}
```

This method:

- Input adapters list all detected SBOMs
- Output adapters show where SBOMs would be sent or saved

✅ Summary

By implementing these six methods, you can fully integrate a new adapter into sbommv—whether it’s for fetching from a custom source, uploading to a private platform, or supporting new SBOM delivery mechanisms.

Adapters let sbommv scale across ecosystems without changing core logic. Stick to the shared interface, and the rest of the pipeline just works.
