# 🔹 Github --> DependencyTrack(dtrack) Examples 🔹

Fetch SBOM from Github System(adapter) and upload it to DependencyTrack System(adapter)

It covers 4 section:

- 

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

## 4. Continuous Monitoring (Daemon Mode): GitHub → DependencyTrack

Enable continuous monitoring by adding the `--daemon` or `-d` flag to your command. In daemon mode, `sbommv` periodically checks for new releases or SBOM updates in the specified GitHub repositories and uploads them to DependencyTrack. The polling interval can be customized using `--in-github-poll-interval`, default poll period is 24hrs.

### 4.1 Single Repository Monitoring

#### 4.1.1 GitHub Release Method (Daemon Mode)

```bash
sbommv transfer --input-adapter=github --in-github-url="https://github.com/interlynk-io/sbomqs" \
                --in-github-method=release --output-adapter=dtrack --out-dtrack-url="http://localhost:8081" \
                --daemon --in-github-poll-interval="60s"
```

**What this does:**

- Continuously monitors the `interlynk-io/sbomqs` repository for new releases containing SBOMs.
- Polls every 60 second (customizable via `--in-github-poll-interval`, supports formats like 60s, 1m, 1hr, or plain seconds).
- Fetches SBOMs from the GitHub Release page when new release is detected.
- Dtrack client automatically creates a new project with name `interlynk-io/sbomqs-<version>-<sbom_file_name>` with project version latest.
- Uploads new SBOMs to the project `interlynk-io/sbomqs-<version>-<sbom_file_name>`.

**NOTE**:

- Use `--in-github-asset-wait-delay` (e.g., `--in-github-asset-wait-delay="180s"`) to add a delay before fetching assets, ensuring GitHub has time to process new releases. By default it take approx 3 minute to release assets after publishing release.
- Cache files (e.g., .`sbommv/cache_dtrack_release.db`) are created to track processed releases and SBOMs, avoiding duplicates.
- To stop the daemon, press `Ctrl+C`.

#### 4.1.2 GitHub API Method (Daemon Mode)

```bash
sbommv transfer --input-adapter=github --in-github-url="https://github.com/interlynk-io/sbomqs" \
                --output-adapter=dtrack --out-dtrack-url="http://localhost:8081" \
                --daemon --in-github-poll-interval="24h"
```

**What this does:**

- Continuously monitors the `interlynk-io/sbomqs` repository for updates to its Dependency Graph SBOM.
- Polls every 24 hours (customizable via `--in-github-poll-interval`).
- Fetches SBOMs using GitHub’s Dependency Graph API when updates are detected.
- Dtrack client automatically creates a new project with name `interlynk-io/sbomqs-latest-dependency-graph-sbom.json` with project version latest.
- Uploads new SBOMs to the project sigstore/cosign.

**NOTE**:

- Cache files (e.g., `.sbommv/cache_dtrack_api.db`) are created to track processed SBOMs, avoiding duplicates.
- To stop the daemon, press Ctrl+C.

#### 4.1.3 GitHub Tool Method (Daemon Mode)

```bash
sbommv transfer --input-adapter=github --in-github-url="https://github.com/interlynk-io/sbomqs" \
                --in-github-method=tool --output-adapter=dtrack --out-dtrack-url="http://localhost:8081" \
                --daemon --in-github-poll-interval="24h"
```

What this does:

- Continuously monitors the `interlynk-io/sbomqs` repository for new releases.
- Polls every 24 hours (customizable via `--in-github-poll-interval`).
- Clones the repository and generates an SBOM using Syft when a new release is detected.
- Dtrack client automatically creates a new project with name `interlynk-io/sbomqs-latest-syft-generated-sbom.json` with project version latest.
- Uploads new SBOMs to the project `interlynk-io/sbomqs-latest-syft-generated-sbom.json`.

NOTE:

- Use `--in-github-branch` (e.g., --in-github-branch="main") to monitor a specific branch.
- Cache files (e.g., `.sbommv/cache_dtrack_tool.db`) are created to track processed releases and SBOMs.
- To stop the daemon, press Ctrl+C.

### 4.2 Multiple Repository Monitoring (Organization-Level)

#### 4.2.1 GitHub Release Method (Daemon Mode)

```bash
sbommv transfer --input-adapter=github --in-github-url="https://github.com/interlynk-io" \
                --in-github-method=release --in-github-include-repos=sbomqs,sbommv \
                --output-adapter=dtrack --out-dtrack-url="http://localhost:8081" \
                --daemon --in-github-poll-interval="24h"
```

**What this does:**

- Continuously monitors the `sbomqs` and `sbommv` repositories under the `interlynk-io` organization for new releases containing SBOMs artifacts..
- Polls every 24 hours (customizable via `--in-github-poll-interval`).
- Fetches SBOMs from the GitHub Release pages when new releases are detected.
- Dtrack client automatically creates projects with names `interlynk-io/sbomqs-<version>-<sbom_file_name>` and `interlynk-io/sbommv-<version>-<sbom_file_name>` with project version latest.
- Uploads new SBOMs to the respective projects.

**NOTE**:

- Use --in-github-exclude-repos (e.g., --in-github-exclude-repos=docs) to exclude specific repositories.
- Cache files (e.g., .sbommv/cache_dtrack_release.db) are created per adapter-method combination.
- To stop the daemon, press Ctrl+C.

#### 4.2.2 GitHub API Method (Daemon Mode)

```bash
sbommv transfer --input-adapter=github --in-github-url="https://github.com/interlynk-io" \
                --in-github-method=api --in-github-include-repos=sbomqs,sbommv \
                --output-adapter=dtrack --out-dtrack-url="http://localhost:8081" \
                --daemon --in-github-poll-interval="24h"
```

**What this does:**

- Continuously monitors the `sbomqs` and `sbommv` repositories under the `interlynk-io` organization for Dependency Graph SBOM updates.
- Polls every 24 hours (customizable via `--in-github-poll-interval`).
- Fetches SBOMs using GitHub’s Dependency Graph API when updates are detected..
- Dtrack client automatically creates projects with names `interlynk-io/sbomqs-latest-dependency-graph-sbom.json` and `interlynk-io/sbommv-latest-dependency-graph-sbom.json` with project version latest.
- Uploads new SBOMs to the respective projects.

**NOTE**:

- Use `--in-github-exclude-repos` (e.g., --in-github-exclude-repos=docs) to exclude specific repositories.
- Cache files (e.g., `.sbommv/cache_dtrack_api.db`) are created per adapter-method combination.
- To stop the daemon, press Ctrl+C.

#### 4.2.3 GitHub Tool Method (Daemon Mode)

```bash
sbommv transfer --input-adapter=github --in-github-url="https://github.com/interlynk-io" \
                --in-github-method=tool --in-github-include-repos=sbomqs,sbommv \
                --output-adapter=dtrack --out-dtrack-url="http://localhost:8081" \
                --daemon --in-github-poll-interval="24h"
```

**What this does:**

- Continuously monitors the `sbomqs` and `sbommv` repositories under the `interlynk-io` organization for Dependency Graph SBOM updates.
- Polls every 24 hours (customizable via `--in-github-poll-interval`).
- Clones the repositories and generates SBOMs using Syft when new releases are detected.
- Dtrack client automatically creates projects with names `interlynk-io/sbomqs-latest-syft-generated-sbom.json` and `interlynk-io/sbommv-latest-syft-generated-sbom.json` with project version latest.
- Uploads new SBOMs to the respective projects.

**NOTE**:

- Cache files (e.g., `.sbommv/cache_dtrack_tool.db`) are created per adapter-method combination.
- To stop the daemon, press Ctrl+C.

## Conclusion

These examples cover various ways to fetch and upload SBOMs using sbommv. Whether you are performing a single transfer, monitoring a single repository, or continuously monitoring an entire organization, sbommv provides flexibility to handle it efficiently.
