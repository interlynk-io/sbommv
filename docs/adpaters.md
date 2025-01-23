# Adapters

## Why sbommv need Adapters ?

- In the sbommv project, we aim to design and implement a robust system of input adapters to handle the retrieval or generation of SBOMs from diverse sources. These sources are:
  - GitHub (by scanning releases, utilizing GitHub APIs, or generating SBOMs using tools like cdxgen), 
  - local folders containing SBOM files,
  - individual SBOM files,
  - AWS S3 buckets,
  - the Interlynk platform (via project ID), and
  - Dependency Track (via project ID)

- To achieve this, we want to ensure the input adapters are modular, adhere to best practices in software design (such as the adapter pattern), and integrate seamlessly with intermediary conversion layers for format handling and output adapters for SBOM transfer.

## Understanding Adapters

The adapter design pattern is used to make incompatible interfaces compatible. It allows your program to use different types of input (or external systems) in a consistent way.

Here’s the general flow of how an adapter works:

- **Client**: The part of your code that needs data or functionality (e.g., sbommv's core logic).
- **Target Interface**: Defines the expected methods or behavior (e.g., InputAdapter).
- **Adapter**: Implements the Target Interface by bridging it to an incompatible external system or library (e.g., GitHub API, local folder scanning, etc.).

## sbommv Input Adapters

Each input adapter should:

- **Retrieve SBOMs**: From a respective source (GitHub, S3, Folder, Interlynk, dTrack, etc.).
- **Expose a Common Interface**: Regardless of the source, all adapters will conform to a shared interface.
- **Be Modular and Independent**: Allow easy addition or replacement of adapters.

## Implementation

InputAdapter defines the interface that all SBOM input adapters must implement

```go
type InputAdapter interface {
 // GetSBOMs retrieves all SBOMs from the source
 GetSBOMs(ctx context.Context) ([]SBOM, error)
}
```


## Examples of Input Adapters

### 1. GitHub Adapter

- **Purpose**: Fetch SBOMs from a **repository’s release page** or **generate them using GitHub’s Dependency Graph API** or **generate via external sbom generating tools**.
- **Key Logic**:
  - Identify SBOM files in releases (already explored previously).
  - Use GitHub’s API to generate an SBOM for the repository.
  - Clone the repository and generate SBOMs using tools like cdxgen.

### 2. Folder Adapter

- **Purpose**:  Scan a local directory for SBOM files.
- Key Logic:
  - Identify valid SBOM files and returns all SBOMs.

### 3. AWS S3 Adapter

- **Purpose**: Fetch SBOMs from an AWS S3 bucket.
- Key Logic:
  - Download and store identified SBOMs locally.

### 4. Interlynk Platform Adapter

- **Purpose**: Retrieve SBOMs from the Interlynk platform based on a project ID.
- **Key Logic**:
  - Use the Interlynk API to fetch SBOMs.
  - And returns all SBOMs from a project ID.

### 6. Dependency Track Adapter

- **Purpose**: Fetch SBOMs from Dependency Track based on a project ID.
- **Key Logic**:
  - Interact with Dependency Track’s API to retrieve SBOMs.
  - Return the fetched SBOMs.
