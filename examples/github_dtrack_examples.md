# 🔹 Github --> DependencyTrack(dtrack) Examples 🔹

Fetch SBOM from Github System(adapter) and upload it to DependencyTrack System(adapter)

## Overview

`sbommv` is a tool designed to transfer SBOMs (Software Bill of Materials) between systems. It operates with two different systems. In this case:

- **Input (Source) System** → Fetches SBOMs  --> **Github**
- **Output (Destination) System** → Uploads SBOMs  --> **DependencyTrack**

### Fetching SBOMs from GitHub Repository

GitHub offers three methods to retrieve/fetch SBOMs from a Repository:

1. **API Method** – Uses GitHub’s [Dependency Graph API](https://docs.github.com/en/enterprise-cloud@latest/rest/dependency-graph/sboms?apiVersion=2022-11-28) to fetch SBOM for a repo.
2. **Release Method** – Extracts SBOMs from Github repository release page.
3. **Tool Method** – Clones the repository and generates SBOMs using Syft  

### Uploading SBOMs to DependencyTrack

Once SBOMs are fetched, they need to be uploaded to DependencyTrack. To setup DependencyTrack, follow this [guide](https://github.com/interlynk-io/sbommv/blob/v0.0.3/examples/setup_dependency_track.md).

## 1. Basic Transfer(Single Repository): GitHub  → DependencyTrack

### 1.1 Github Release Method

```bash
sbommv transfer --input-adapter=github --in-github-url="https://github.com/sigstore/cosign" \
                --in-github-method=release --output-adapter=dtrack --out-dtrack-url="http://localhost:8081"
```

- **What this does**:
  - Fetches SBOMs from GitHub’s Release page of `sigstore/cosign`
  - dtrack client automatically creates a new project with name `sigstore/cosign` with project version `latest`
  - Uploads those sboms to a project `sigstore/cosign`

#### Let's specify a project name and version

```bash
sbommv transfer --input-adapter=github --in-github-url="https://github.com/sigstore/cosign" \
                --in-github-method=release --output-adapter=dtrack --out-dtrack-url="http://localhost:8081" \        
∙               --out-dtrack-project-name="cosign_demo" --out-dtrack-project-version="v1.0.1"
```

- **What this does**:
  - Fetches SBOMs from GitHub’s Release page of `sigstore/cosign`
  - dtrack client creates a new project with name `cosign_demo` with project version `v1.0.1`
  - Uploads those sboms to a project `cosign_demo`

### 1.2 GitHub API Method (Dependency Graph): default method

```bash
sbommv transfer --input-adapter=github --in-github-url="https://github.com/sigstore/cosign" \
                 --output-adapter=dtrack --out-dtrack-url="http://localhost:8081"
```

- **What this does**:
  - Fetches SBOMs from GitHub’s dependency graph API for the repository `sigstore/cosign`
  - dtrack client automatically creates a new project with name `sigstore/cosign` with project version `latest`
  - Uploads those sboms to a project `sigstore/cosign`

**NOTE**:

- Best for repositories that don’t publish SBOMs in releases.

### 1.3 GitHub Tool Method (SBOM Generation Using Syft)

```bash
sbommv transfer --input-adapter=github --in-github-url="https://github.com/sigstore/cosign" \
                --in-github-method=tool --output-adapter=dtrack --out-dtrack-url="http://localhost:8081"
```

- **What this does**:
  - Clones the repository
  - Generates an SBOM using Syft for the repository `sigstore/cosign`
  - dtrack client automatically creates a new project with name `sigstore/cosign` with project version `latest`
  - Uploads those sboms to a project `sigstore/cosign`

**NOTE**:

- Best for repositories without SBOMs in API or releases

#### 1.3.1 Fetch SBOMs for a Specific GitHub Branch (Tool Method Only)

```bash
sbommv transfer --input-adapter=github --in-github-url="https://github.com/sigstore/cosign" \
                --in-github-method=tool --in-github-branch="main" \
                --output-adapter=dtrack --out-dtrack-url="http://localhost:8081"
```

- **What this does**:
  - Clones the main branch instead of the default branch.
  - Generates an SBOM using Syft for the repository `sigstore/cosign`
  - dtrack client automatically creates a new project with name `sigstore/cosign` with project version `latest`
  - Uploads those sboms to a project `sigstore/cosign`

**NOTE**:

- Only applicable to the `tool` method because releases and API do not support branches.

## 2. Using Dry-Run Mode (No Upload, Just Simulation)

```bash
sbommv transfer --input-adapter=github --in-github-url="https://github.com/sigstore/cosign" \
                --output-adapter=dtrack --out-dtrack-url="http://localhost:8081" --dry-run
```

- **What this does**:
  - Fetches SBOMs without uploading (simulates the process).
  - Displays what would be uploaded (preview mode)

**NOTE**:

- Useful for previewing the SBOMs to be uploaded, project to be created on dtrack.
- Useful for testing before actual uploads.

## 3. Advanced Transfer(Multiple Repositories in an Organization)

### 3.1 Include Repos of an Organization

#### 3.1.1 Github Release Method

```bash
sbommv transfer --input-adapter=github --in-github-url="https://github.com/sigstore" \
                --in-github-method=release --in-github-include-repos=cosign,rekor \
                --output-adapter=dtrack --out-dtrack-url="http://localhost:8081"
```

- **What this does**:
  - Fetches SBOMs only from `cosign` and `rekor` repositories under `sigstore` organization
  - dtrack client automatically creates a new project with name `sigstore/cosign`, `sigstore/rekor` with project version `latest`
  - `cosign`, `rekor` SBOMs will be uploaded to `sigstore/cosign` and `sigstore/rekor` respectively.

**NOTE**:

- Use `--in-github-include-repos` to specify which repos to fetch

#### 3.1.2 Github API Method

```bash
sbommv transfer --input-adapter=github --in-github-url="https://github.com/sigstore" \
                --in-github-include-repos=cosign,rekor \
                --output-adapter=dtrack --out-dtrack-url="http://localhost:8081"
```

- **What this does**:
  - Fetches SBOMs only from `cosign` and `rekor` repositories in the `sigstore` organization.
  - dtrack client automatically creates a new project with name `sigstore/cosign`, `sigstore/rekor` with project version `latest`
  - `cosign`, `rekor` SBOMs will be uploaded to `sigstore/cosign` and `sigstore/rekor` respectively.

#### 3.1.3 Github Tool Method

```bash
sbommv transfer --input-adapter=github --in-github-url="https://github.com/sigstore" \
                --in-github-method=tool --in-github-include-repos=cosign,rekor \
                --output-adapter=dtrack --out-dtrack-url="http://localhost:8081"
```

- **What this does**:
  - Fetches SBOMs only from `cosign` and `rekor` repositories in the `sigstore` organization.
  - dtrack client automatically creates a new project with name `sigstore/cosign`, `sigstore/rekor` with project version `latest`
  - `cosign`, `rekor` SBOMs will be uploaded to `sigstore/cosign` and `sigstore/rekor` respectively.

### 3.2 Exclude Certain Repositories from an Organization

#### 3.2.1  Github Release Method

```bash
sbommv transfer --input-adapter=github --in-github-url="https://github.com/sigstore" \
                --in-github-method=release --in-github-exclude-repos=docs \
                --output-adapter=dtrack --out-dtrack-url="http://localhost:8081"
```

- **What this does**:
  - Fetches SBOMs from all repositories under `sigstore` except `docs` from respective Release Page.
  - dtrack client automatically creates a new project with name `sigstore/cosign`, `sigstore/rekor`, and similarly for all others repo present in the `sigstore` organization except `docs` repo.
  - `cosign`, `rekor` and all other repo SBOMs will be uploaded to `sigstore/cosign` , `sigstore/rekor` , etc  respectively.

**NOTE**:

- It only fetches SBOM from repo which contains SBOM in their release page.
- ❌ SBOMs won't be fetched in two cases:
  - The repository has a release page, but no SBOM artifacts (e.g., the release contains binaries but no SBOM JSON file).
  - The repository has no releases at all (i.e., the "Releases" tab doesn’t exist for that repo).
- So, in both case SBOM wouldn't be fetched.

#### 3.2.2 Github API Method

```bash
sbommv transfer --input-adapter=github --in-github-url="https://github.com/sigstore" \
                --in-github-exclude-repos=docs \
                --output-adapter=dtrack --out-dtrack-url="http://localhost:8081"
```

- **What this does**:
  - Fetches SBOMs from all repositories in `sigstore` except `docs`.
  - Using Dependency Graph API.
  - Uploads them as separate projects in dtrack.

**NOTE**:

- ❌ SBOMs won't be fetched in these cases:
  - The repository has no dependency manifest file (e.g., package.json, go.mod, requirements.txt, etc.).
  - The repository owner has disabled Dependency Graph for their repo in GitHub settings.
  - The repository is private, and you don't have the right permissions (the API only works for public repos unless authenticated with the correct permissions).
  - GitHub doesn't support dependency extraction for that particular language (not all package managers are covered).

#### 3.2.3 Github Tool Method

```bash
sbommv transfer --input-adapter=github --in-github-url="https://github.com/sigstore" \
                --in-github-method=tool --in-github-exclude-repos=docs \
                --output-adapter=dtrack --out-dtrack-url="http://localhost:8081"
```

- **What this does**:
  - Fetches SBOMs from all repositories in `sigstore` except `docs`.
  - Uploads them as separate projects in dtrack.

**NOTE**:

- ❌ SBOMs won't be fetched if:
  - The repository is empty (no source code) – Syft needs files to scan.
  - The repository only contains non-code files (e.g., just README.md and documentation).
  - Syft does not support the programming language used in the repo (e.g., Syft primarily supports package managers like npm, pip, go modules, etc.).
  - The repository has large binary files instead of source code – Syft analyzes source-level dependencies, not compiled binaries.
  - The repository is private, and you don’t have access to clone it.

## Conclusion

These examples cover various ways to fetch and upload SBOMs using sbommv. Whether you are fetching SBOMs from a single repo, an entire organization, or using a specific branch, sbommv provides flexibility to handle it efficiently.
