
# 🔹 Github --> Interlynk Examples 🔹

Fetch SBOM from Github System(adapter) and upload it to Interlynk System(adapter)

## Overview

`sbommv` is a tool designed to transfer SBOMs (Software Bill of Materials) between systems. It operates with two different systems. In this case:

- **Input (Source) System** → Fetches SBOMs  --> **github**
- **Output (Destination) System** → Uploads SBOMs  --> **Interlynk**

### Fetching SBOMs from GitHub Repository

GitHub offers three methods to retrieve/fetch SBOMs from a Repository:

1. **API Method** – Uses GitHub’s [Dependency Graph API](https://docs.github.com/en/enterprise-cloud@latest/rest/dependency-graph/sboms?apiVersion=2022-11-28) to fetch SBOM for a repo.
2. **Release Method** – Extracts SBOMs from Github repository release page.
3. **Tool Method** – Clones the repository and generates SBOMs using Syft  

### Uploading SBOMs to Interlynk

Once SBOMs are fetched, they need to be uploaded to Interlynk. To use Interlynk, you need to:

1. [Create an Interlynk account](https://app.interlynk.io/auth).
2. Generate an **INTERLYNK_SECURITY_TOKEN** from [here](https://app.interlynk.io/vendor/settings?tab=security%20tokens).
3. Export the token before running `sbommv`

    ```bash
    export INTERLYNK_SECURITY_TOKEN="lynk_test_EV2DxRfCfn4wdM8FVaiGkb6ny3KgSJ7JE5zT"
    ```

Now let's dive into various use cases and examples.

## 1. Basic Transfer(Single Repository): GitHub  → Interlynk

### 1.1 Github Release Method

#### Fetch SBOMs from the latest release of a Github repository and upload to Interlynk

```bash
sbommv transfer --input-adapter=github --in-github-url="https://github.com/sigstore/cosign" \
                --in-github-method=release --output-adapter=interlynk --out-interlynk-url="https://api.interlynk.io/lynkapi" --dry-run
```

- **What this does**:
  - Fetches SBOMs from the latest release of the repository `sigstore/cosign`
  - And prints the feched SBOMs as well as SBOMs to be uploaded on Interlynk
  
- **NOTE**:
  - `dry-run` method display the preview of the SBOMs to be uploaded, project to be created on Interlynk.
  - remove the `dry-run` flag, to process with uploading part.

```bash
sbommv transfer --input-adapter=github --in-github-url="https://github.com/sigstore/cosign" \
                --in-github-method=release --output-adapter=interlynk --out-interlynk-url="https://api.interlynk.io/lynkapi"
```

### 1.2 GitHub API Method (Dependency Graph): default method

#### Fetch SBOMs using the GitHub API and upload to Interlynk

```bash
sbommv transfer --input-adapter=github --in-github-url="https://github.com/sigstore/cosign" \
                 --output-adapter=interlynk --out-interlynk-url="https://api.interlynk.io/lynkapi"
```

- **What this does**:
  - Fetches SBOMs from GitHub’s dependency graph API for the repository `sigstore/cosign`
  - Uploads them to Interlynk

**NOTE**:

- Best for repositories that don’t publish SBOMs in releases.

### 1.3 GitHub Tool Method (SBOM Generation Using Syft)

#### Clone the repo, generate an SBOM using Syft, and upload it to Interlynk.

```bash
sbommv transfer --input-adapter=github --in-github-url="https://github.com/sigstore/cosign" \
                --in-github-method=tool --output-adapter=interlynk --out-interlynk-url="https://api.interlynk.io/lynkapi"
```

- **What this does**:
  - Clones the repository
  - Generates an SBOM using Syft for the repository `sigstore/cosign`
  - Uploads them to Interlynk

**NOTE**:

- Best for repositories without SBOMs in API or releases

#### 1.3.1 Fetch SBOMs for a Specific GitHub Branch (Tool Method Only)

```bash
sbommv transfer --input-adapter=github --in-github-url="https://github.com/sigstore/cosign" \
                --in-github-method=tool --in-github-branch="main" \
                --output-adapter=interlynk --out-interlynk-url="https://api.interlynk.io/lynkapi"
```

- **What this does**:
  - Clones the main branch instead of the default branch.
  - Generates an SBOM using Syft for the repository `sigstore/cosign`
  - Uploads them to Interlynk

**NOTE**:

- Only applicable to the `tool` method because releases and API do not support branches.

## 2. Using Dry-Run Mode (No Upload, Just Simulation)

```bash
sbommv transfer --input-adapter=github --in-github-url="https://github.com/sigstore/cosign" \
                --output-adapter=interlynk --out-interlynk-url="https://api.interlynk.io/lynkapi" --dry-run
```

- **What this does**:
  - Fetches SBOMs without uploading (simulates the process).
  - Displays what would be uploaded (preview mode)

**NOTE**:

- Useful for previewing the SBOMs to be uploaded, project to be created on Interlynk.
- Useful for testing before actual uploads.

## 3. Advanced Transfer(Multiple Repositories in an Organization)

### 3.1 Include Repos of an Organization

#### 3.1.1 Github Release Method

```bash
sbommv transfer --input-adapter=github --in-github-url="https://github.com/sigstore" \
                --in-github-method=release --in-github-include-repos=cosign,rekor \
                --output-adapter=interlynk --out-interlynk-url="https://api.interlynk.io/lynkapi"
```

- **What this does**:
  - Fetches SBOMs only from `cosign` and `rekor` repositories under `sigstore` organization
  - Uploads them as separate projects in Interlynk.
  - `cosign`, `rekor` SBOMs will be uploaded to `sigstore/cosign` and `sigstore/rekor` respectively.

**NOTE**:

- Use `--in-github-include-repos` to specify which repos to fetch

#### 3.1.2 Github API Method

```bash
sbommv transfer --input-adapter=github --in-github-url="https://github.com/sigstore" \
                --in-github-include-repos=cosign,rekor \
                --output-adapter=interlynk --out-interlynk-url="https://api.interlynk.io/lynkapi"
```

- **What this does**:
  - Fetches SBOMs only from `cosign` and `rekor` repositories in the `sigstore` organization.
  - Uploads them as separate projects in Interlynk.
  - `cosign`, `rekor` SBOMs will be uploaded to `sigstore/cosign` and `sigstore/rekor` respectively.

#### 3.1.3 Github Tool Method

```bash
sbommv transfer --input-adapter=github --in-github-url="https://github.com/sigstore" \
                --in-github-method=tool --in-github-include-repos=cosign,rekor \
                --output-adapter=interlynk --out-interlynk-url="https://api.interlynk.io/lynkapi"
```

- **What this does**:
  - Fetches SBOMs only from `cosign` and `rekor` repositories in the `sigstore` organization.
  - Uploads them as separate projects in Interlynk.
  - `cosign`, `rekor` SBOMs will be uploaded to `sigstore/cosign` and `sigstore/reko` respectively.

### 3.2 Exclude Certain Repositories from an Organization

#### 3.2.1  Github Release Method

```bash
sbommv transfer --input-adapter=github --in-github-url="https://github.com/sigstore" \
                --in-github-method=release --in-github-exclude-repos=docs \
                --output-adapter=interlynk --out-interlynk-url="https://api.interlynk.io/lynkapi"
```

- **What this does**:
  - Fetches SBOMs from all repositories in `sigstore` except `docs`.
  - From Release Page.
  - Uploads them as separate projects in Interlynk.

#### 3.2.2 Github API Method

```bash
sbommv transfer --input-adapter=github --in-github-url="https://github.com/sigstore" \
                --in-github-exclude-repos=docs \
                --output-adapter=interlynk --out-interlynk-url="https://api.interlynk.io/lynkapi"
```

- **What this does**:
  - Fetches SBOMs from all repositories in `sigstore` except `docs`.
  - Using Dependency Graph API.
  - Uploads them as separate projects in Interlynk.

#### 3.2.3 Github Tool Method

```bash
sbommv transfer --input-adapter=github --in-github-url="https://github.com/sigstore" \
                --in-github-method=tool --in-github-exclude-repos=docs \
                --output-adapter=interlynk --out-interlynk-url="https://api.interlynk.io/lynkapi"
```

- **What this does**:
  - Fetches SBOMs from all repositories in `sigstore` except `docs`.
  - Uploads them as separate projects in Interlynk.

## 4. GitHub  → Folder

### Fetch SBOMs from github repo and save it to a folder

```bash
 sbommv transfer --input-adapter=github --in-github-url="https://github.com/sigstore/" --in-github-include-repos=cosign,fulcio,rekor --in-github-method="release" --output-adapter=folder --out-folder-path="temp"
```

- **What this does**:
  - Fetches SBOMs from `sigstore` organization for repositories in `cosign`, `fulcio`, and `rekor`.
  - Save these SBOMs to a folder `temp`.
  - Under `temp`, seperate sub-dir with name `cosign`, `fulcio` and `rekor` will be created and respective repo SBOMs will be stored there.

## 4. Folder → Interlynk

### Fetch SBOMs from folder "temp" and upload/push it to a Interlynk

```bash
sbommv transfer --input-adapter=folder --in-folder-path="temp"  --in-folder-recursive=true  --output-adapter=interlynk --out-interlynk-url="http://localhost:3000/lynkapi"
```

- **What this does**:
  - Fetches/Scan SBOMs from `temp` directory for all sub-directories such as `cosign`, `fulcio`, and `rekor`.
  - Upload these SBOMs to a Interlynk with a project ID `cosign`, `fulcio`, and `rekor`.

## Some More Examples

### Combine Multiple Flags for Full Customization

```bash
sbommv transfer --input-adapter=github --in-github-url="https://github.com/sigstore/cosign" \
                --in-github-method=tool --in-github-branch="dev" \
                --output-adapter=interlynk --out-interlynk-url="https://api.interlynk.io/lynkapi" \
                --out-interlynk-project-name="cosign-dev" --out-interlynk-project-env="development"
```

- **What This Does**:
  - Fetches SBOMs using tool from cosign for dev branch
  - Uploads them to a specific Interlynk project (`cosign-dev`)
  - Uses the `development` environment instead of the default

NOTE:

- Project `cosign-dev` must be present in the Interlynk

## Conclusion

These examples cover various ways to fetch and upload SBOMs using sbommv. Whether you are fetching SBOMs from a single repo, an entire organization, or using a specific branch, sbommv provides flexibility to handle it efficiently.
