
# 🔹Github --> Folder Examples 🔹

Fetch SBOM from Github System(adapter) and saves it to Folder System(adapter)

## Overview  

`sbommv` is a tool designed to transfer SBOMs (Software Bill of Materials) between systems. It operates with two different systems. In this case:

- **Input (Source) System** → Fetches SBOMs  --> **Github**
- **Output (Destination) System** → Uploads SBOMs  --> **Folder**

### Fetching SBOMs from GitHub Repository

GitHub offers three methods to retrieve/fetch SBOMs from a Repository:

1. **API Method** – Uses GitHub’s [Dependency Graph API](https://docs.github.com/en/enterprise-cloud@latest/rest/dependency-graph/sboms?apiVersion=2022-11-28) to fetch SBOM for a repo.
2. **Release Method** – Extracts SBOMs from Github repository release page.
3. **Tool Method** – Clones the repository and generates SBOMs using Syft  

### Saves SBOMs to local Folder

Once SBOMs are fetched, they need to be saved to Folder. To use Folder, you need to provide the **path of the folder**.

Now let's dive into various use cases and examples.

## 1. Basic Transfer(Single Repository): GitHub  → Folder

### 1.1 Github Release Method

```bash
sbommv transfer --input-adapter=github --in-github-url="https://github.com/sigstore/cosign" \
                --in-github-method=release --output-adapter=folder --out-folder-path="temp-release" --dry-run
```

- **What this does**:
  - Fetches SBOMs from the latest release of the repository `sigstore/cosign`
  - And saves the feched SBOMs to a `temp-release` folder.
  
- **NOTE**:
  - `dry-run` method display the preview of the SBOMs to be uploaded, project to be created on Interlynk.
  - remove the `dry-run` flag, to process with uploading part.

```bash
sbommv transfer --input-adapter=github --in-github-url="https://github.com/sigstore/cosign" \
                --in-github-method=release --output-adapter=folder --out-folder-path="temp-release"
```

### 1.2 GitHub API Method (Dependency Graph): default method

```bash
sbommv transfer --input-adapter=github --in-github-url="https://github.com/sigstore/cosign" \
                 --output-adapter=folder --out-folder-path="temp-api"
```

- **What this does**:
  - Fetches SBOMs from GitHub’s dependency graph API for the repository `sigstore/cosign`
  - Save them to `temp-api` folder.

**NOTE**:

- Best for repositories that don’t publish SBOMs in releases.

### 1.3 GitHub Tool Method (SBOM Generation Using Syft)

```bash
sbommv transfer --input-adapter=github --in-github-url="https://github.com/sigstore/cosign" \
                --in-github-method=tool --output-adapter=folder --out-folder-path="temp-tool"
```

- **What this does**:
  - Clones the repository
  - Generates an SBOM using Syft for the repository `sigstore/cosign`
  - Save to `temp-tool` folder.

**NOTE**:

- Best for repositories without SBOMs in API or releases

### 1.4 Fetch SBOMs for a Specific GitHub Branch (GitHub Tool Method Only)

```bash
sbommv transfer --input-adapter=github --in-github-url="https://github.com/sigstore/cosign" \
                --in-github-method=tool --in-github-branch="main" \
                --output-adapter=folder --out-folder-path="temp-tool-branch"
```

- **What this does**:
  - Clones the main branch instead of the default branch.
  - Generates an SBOM using Syft for the repository `sigstore/cosign`
  - Saves them to `temp-tool-branch` folder

**NOTE**:

- Only applicable to the `tool` method because releases and API do not support branches.

## 2. Using Dry-Run Mode (No Upload, Just Simulation)

```bash
sbommv transfer --input-adapter=github --in-github-url="https://github.com/sigstore/cosign" \
                --output-adapter=folder --out-folder-path="temp" --dry-run
```

- **What this does**:
  - Fetches SBOMs without uploading (simulates the process).
  - Displays what would be saved (preview mode)

**NOTE**:

- Useful for previewing the SBOMs to be saved, sub-folder will be created for each repo.
- Useful for testing before actual saved.

## 3. Advanced Transfer(Multiple Repositories in an Organization)

### 3.1 Include Repos of an Organization

#### 3.1.1 Github Release Method

```bash
sbommv transfer --input-adapter=github --in-github-url="https://github.com/sigstore" \
                --in-github-method=release --in-github-include-repos=cosign,rekor \
                --output-adapter=folder --out-folder-path="temp-release"
```

- **What this does**:
  - Fetches SBOMs only from `cosign` and `rekor` repositories under `sigstore` organization
  - Save them as separate sub-dir inside `temp-release` folder.
  - `cosign`, `rekor` SBOMs will be saved as `temp-release/cosign` and `temp-release/rekor` respectively.

**NOTE**:

- Use `--in-github-include-repos` to specify which repos to fetch

#### 3.1.2 Github API Method

```bash
sbommv transfer --input-adapter=github --in-github-url="https://github.com/sigstore" \
                --in-github-include-repos=cosign,rekor \
                --output-adapter=folder --out-folder-path="temp-api"
```

- **What this does**:
  - Fetches SBOMs only from `cosign` and `rekor` repositories in the `sigstore` organization.
  - Save them as sub-dir inside `temp-api` folder.
  - `cosign`, `rekor` SBOMs will be saved to `temp-api/cosign` and `temp-api/rekor` respectively.

#### 3.1.3 Github Tool Method

```bash
sbommv transfer --input-adapter=github --in-github-url="https://github.com/sigstore" \
                --in-github-method=tool --in-github-include-repos=cosign,rekor \
                --output-adapter=folder --out-folder-path="temp-tool"
```

- **What this does**:
  - Fetches SBOMs only from `cosign` and `rekor` repositories in the `sigstore` organization.
  - Save them as sub-dir inside `temp-tool` folder.
  - `cosign`, `rekor` SBOMs will be saved to `temp-tool/cosign` and `temp-tool/rekor` respectively.

### 3.2 Exclude Certain Repositories from an Organization

#### 3.2.1  Github Release Method

```bash
sbommv transfer --input-adapter=github --in-github-url="https://github.com/sigstore" \
                --in-github-method=release --in-github-exclude-repos=docs \
                --output-adapter=folder --out-folder-path="temp-release"
```

- **What this does**:
  - Fetches SBOMs from all repositories in `sigstore` except `docs`.
  - From Release Page.
  - Save them as sub-dir inside `temp-release` folder.

#### 3.2.2 Github API Method

```bash
sbommv transfer --input-adapter=github --in-github-url="https://github.com/sigstore" \
                --in-github-exclude-repos=docs \
                --output-adapter=folder --out-folder-path="temp-api"
```

- **What this does**:
  - Fetches SBOMs from all repositories in `sigstore` except `docs`.
  - Using Dependency Graph API.
  - Save them as sub-dir inside `temp-api` folder.

#### 3.2.3 Github Tool Method

```bash
sbommv transfer --input-adapter=github --in-github-url="https://github.com/sigstore" \
                --in-github-method=tool --in-github-exclude-repos=docs \
                --output-adapter=folder --out-folder-path="temp-tool"
```

- **What this does**:
  - Fetches SBOMs from all repositories in `sigstore` except `docs`.
  - Save them as sub-dir inside `temp-tool` folder.

## Conclusion

These examples cover various ways to fetch and upload SBOMs using sbommv. Whether you are fetching SBOMs from a single repo, an entire organization, or using a specific branch, sbommv provides flexibility to handle it efficiently.
