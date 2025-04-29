# Adapters

## Why Does sbommv Use Adapters?

sbommv is designed to automate the transfer of SBOMs across systems—from where they’re generated to where they’re analyzed or stored. To support a wide variety of SBOM sources and destinations systems, it uses a modular adapter-based architecture.

Adapter allows sbommv to interact with external systems like GitHub, local folders, AWS S3, Interlynk, and Dependency-Track—
without embedding system specific logic in the core engine. This approach keeps the architecture clean, extensible and easy to maintain, while enabling future support for additional systems based on evolving usecases and requirements.

To make this work, all adapter follows a common interface, adhere to software design best practices(such as the adapter pattern), and integrate seamlessly with conversion layers to handle format translation and according to the acceptability of SBOMs of output or destination systems.

## Understanding Adapters from sbommv perspective

An adapter in a sbommv is a pluggable component responsible for interacting with a specific system—whether it’s fetching SBOMs from a source or uploading them to a destination. sbommv follows the adapter design pattern, which decouples system-specific logic from the core engine and ensures all adapters conform to a unified interface.

## Adapter Interface

All adapters must implement the following interface:

```go
type Adapter interface {
	AddCommandParams(cmd *cobra.Command)
	ParseAndValidateParams(cmd *cobra.Command) error
	FetchSBOMs(ctx tcontext.TransferMetadata) (iterator.SBOMIterator, error)
	UploadSBOMs(ctx tcontext.TransferMetadata, iterator iterator.SBOMIterator) error
	DryRun(ctx tcontext.TransferMetadata, iterator iterator.SBOMIterator) error
}
```

### General Flow of Adapters in sbommv

#### 1. Parameter Handling

Each adapter defines it's own CLI flags and arguments via `AddCommandParams`, allowing users to configure inputs and outputs per system.

#### 2. Validation

Adapters validate the provided arguments using `ParseAndValidateParams` before initiating any action.

#### 3. Fetching SBOMs (Input Adapter)

The selected input adapter lazily retrieves SBOMs using the `FetchSBOMs` method. This returns an iterator, allowing sbommv to process large sets of SBOMs efficiently and in a streaming fashion.

#### 4. SBOM Processing Layer

This layer is triggered only when required by the destination system. For example, Dependency-Track only supports SBOMs in CycloneDX format, so sbommv automatically converts SBOMs from formats like SPDX to CycloneDX as part of the upload process.

Internally, sbommv leverages the protobom library to handle format conversion and normalization. This ensures compatibility without requiring the user to pre-process SBOMs manually.

#### 5. Uploading SBOMs (Output Adapter)

The output adapter takes the processed SBOMs and sends them to the target system using `UploadSBOMs`. Depending on the destination, this could involve pushing to an API, writing to disk, or uploading to cloud storage.

#### 6. Dry-Run Support

Both input and output adapters optionally implement `DryRun` to simulate the transfer and display what would be fetched or uploaded—without actually performing any operation.

## Supported Adapters

### Input Adapters

Adapters responsible for retrieving SBOMs:

#### 1. GitHub Adapter

Fetches or generates SBOMs from GitHub repositories:

- **API method** – Uses GitHub’s Dependency Graph API (SPDX format only).
- **Release metho**d – Downloads SBOMs attached to GitHub Releases.
- **Tool method** – Clones the repo and generates SBOMs using tools like syft.

#### 2. Folder Adapter

Scans a local directory and returns valid SBOMs for processing.

#### 3. AWS S3 Adapter

Downloads valid SBOMs from an S3 bucket and returns them for transfer.

#### 4. Interlynk Adapter (Upcoming)

Fetches SBOMs from the Interlynk platform using a project ID.

#### 5. Dependency-Track Adapter (Upcoming)

Fetches SBOMs from Dependency-Track using a project UUID.

### Output Adapters

Adapters responsible for uploading SBOMs:

#### 1. Folder Adapter

Stores SBOMs into a specified local directory.

#### 2. Dependency-Track Adapter

Pushes SBOMs to Dependency-Track instance.

#### 3. Interlynk Adapter

Uploads SBOMs to Interlynk platform.

#### 4. AWS S3 Adapter

Upload SBOMs to AWS S3 cloud storage.

## Wrapping Up

Adapters are at the heart of sbommv’s flexibility. By abstracting how SBOMs are retrieved and where they are sent, sbommv provides a clean and scalable way to manage SBOM movement between systems. As the SBOM ecosystem continues to grow, this modular approach ensures sbommv can evolve with it—supporting more sources, formats, and platforms, without sacrificing maintainability.

Whether you're integrating with GitHub, cloud storage, local tools, or enterprise platforms—sbommv adapters handle the complexity so you don’t have to. If you are looking to integrate own system for seamlessly flow, open a feature request via [issue](https://github.com/interlynk-io/sbommv/issues/new).
