
# 🔹 SBOMMV Transfer Examples 🔹

## **Overview**  

`sbommv` is a tool designed to transfer SBOMs (Software Bill of Materials) between systems. It operates with two types of systems:  

- **Input (Source) System** → Fetches SBOMs  
- **Output (Destination) System** → Uploads SBOMs  

Currently, `sbommv` supports **GitHub** as an input system and **Interlynk** as an output system.  

### **Fetching SBOMs from GitHub**  

GitHub offers three methods to retrieve SBOMs:

1. **API Method** – Uses GitHub’s Dependency Graph  
2. **Release Method** – Extracts SBOMs from GitHub releases  
3. **Tool Method** – Clones the repository and generates SBOMs using Syft  

### **Uploading SBOMs to Interlynk**

Once SBOMs are fetched, they are uploaded to Interlynk. To use Interlynk, you need to:

1. [Create an Interlynk account](https://app.interlynk.io/auth).
2. Generate an INTERLYNK_SECURITY_TOKEN from [here](https://app.interlynk.io/vendor/settings?tab=security%20tokens).
3. Export the token** before running `sbommv`
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
  - Fetches SBOMs from the latest release of the repository sigstore/cosign
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
  - Fetches SBOMs from GitHub’s dependency graph API for the repository sigstore/cosign
  - Uploads them to Interlynk

**NOTE**:

- This method is useful when no SBOMs are published in releases

### 1.3 GitHub Tool Method (SBOM Generation Using Syft)

#### Fetch SBOMs by cloning the repo, generating an SBOM with Syft, and uploading it to Interlynk

```bash
sbommv transfer --input-adapter=github --in-github-url="https://github.com/sigstore/cosign" \
                --in-github-method=tool --output-adapter=interlynk --out-interlynk-url="https://api.interlynk.io/lynkapi"
```

- **What this does**:
  - Clones the repository
  - Generates an SBOM using Syft for the repository sigstore/cosign
  - Uploads them to Interlynk

**NOTE**:

- Useful when neither API nor Release provides SBOMs.

#### 1.3.1 Fetch SBOMs for a Specific GitHub Branch (Tool Method Only)

```bash
sbommv transfer --input-adapter=github --in-github-url="https://github.com/sigstore/cosign" \
                --in-github-method=tool --in-github-branch="main" \
                --output-adapter=interlynk --out-interlynk-url="https://api.interlynk.io/lynkapi"
```

- **What this does**:
  - Clones the main branch instead of the default branch.
  - Generates an SBOM using Syft for the repository sigstore/cosign
  - Uploads them to Interlynk

**NOTE**:

- Only applicable to the `tool` method because releases and API do not support branches.

## 2. Using Dry-Run Mode (No Upload, Just Simulation)**

```bash
sbommv transfer --input-adapter=github --in-github-url="https://github.com/sigstore/cosign" \
                --output-adapter=interlynk --out-interlynk-url="https://api.interlynk.io/lynkapi" --dry-run
```

- **What this does**:
  - Fetches SBOMs but does not upload them
  - Displays what would be uploaded (preview mode)

**NOTE**:

- Useful for previewing the SBOMs to be uploaded, project to be created on Interlynk.

## 2. Advanced Transfer(Organization Repos): GitHub → Interlynk

### 2.1 Github Release Method

#### Fetch SBOMs from a GitHub Organization, Including Only Specific Repos and then upload to interlynk

```bash
sbommv transfer --input-adapter=github --in-github-url="https://github.com/sigstore" \
                --in-github-method=release --in-github-include-repos=cosign,rekor \
                --output-adapter=interlynk --out-interlynk-url="https://api.interlynk.io/lynkapi"
```

- **What this does**:
  - Fetches SBOMs only from `cosign` and `rekor` repositories in the `sigstore` organization
  - Uploads them as separate projects in Interlynk.
  - `cosign`, `rekor` SBOMs will be uploaded to `sigstore/cosign` and `sigstore/reko` respectively.

**NOTE**:

- Use --in-github-include-repos to specify which repos to fetch

#### Fetch SBOMs from a GitHub Organization, Excluding Certain Repos and then upload to interlynk

```bash
sbommv transfer --input-adapter=github --in-github-url="https://github.com/sigstore" \
                --in-github-method=release --in-github-exclude-repos=docs \
                --output-adapter=interlynk --out-interlynk-url="https://api.interlynk.io/lynkapi"
```

- **What this does**:
  - Fetches SBOMs from all repositories in `sigstore` except `docs`.
  - Uploads them as separate projects in Interlynk.

### 2.2 Github API Method

#### Fetch SBOMs from a GitHub Organization by including Specific Repos and then upload to interlynk

```bash
sbommv transfer --input-adapter=github --in-github-url="https://github.com/sigstore" \
                --in-github-include-repos=cosign,rekor \
                --output-adapter=interlynk --out-interlynk-url="https://api.interlynk.io/lynkapi"
```

- **What this does**:
  - Fetches SBOMs only from `cosign` and `rekor` repositories in the `sigstore` organization.
  - Uploads them as separate projects in Interlynk.
  - `cosign`, `rekor` SBOMs will be uploaded to `sigstore/cosign` and `sigstore/reko` respectively.


#### Fetch SBOMs using API method from a GitHub Organization by excluding Specific Repos and then upload to interlynk

```bash
sbommv transfer --input-adapter=github --in-github-url="https://github.com/sigstore" \
                --in-github-exclude-repos=docs \
                --output-adapter=interlynk --out-interlynk-url="https://api.interlynk.io/lynkapi"
```

- **What this does**:
  - Fetches SBOMs from all repositories in `sigstore` except `docs`.
  - Uploads them as separate projects in Interlynk.

### 2.3 Github Tool Method

#### Fetch SBOMs from a GitHub Organization by including Specific Repos and then upload to interlynk

```bash
sbommv transfer --input-adapter=github --in-github-url="https://github.com/sigstore" \
                --in-github-method=tool --in-github-include-repos=cosign,rekor \
                --output-adapter=interlynk --out-interlynk-url="https://api.interlynk.io/lynkapi"
```

- **What this does**:
  - Fetches SBOMs only from `cosign` and `rekor` repositories in the `sigstore` organization.
  - Uploads them as separate projects in Interlynk.
  - `cosign`, `rekor` SBOMs will be uploaded to `sigstore/cosign` and `sigstore/reko` respectively.

#### Fetch SBOMs using Tool method from a GitHub Organization by excluding Specific Repos and then upload to interlynk

```bash
sbommv transfer --input-adapter=github --in-github-url="https://github.com/sigstore" \
                --in-github-method=tool --in-github-exclude-repos=docs \
                --output-adapter=interlynk --out-interlynk-url="https://api.interlynk.io/lynkapi"
```

- **What this does**:
  - Fetches SBOMs from all repositories in `sigstore` except `docs`.
  - Uploads them as separate projects in Interlynk.

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

- Project `cosign-project` must be present in the Interlynk

## Conclusion

These examples cover various ways to fetch and upload SBOMs using sbommv. Whether you are fetching SBOMs from a single repo, an entire organization, or using a specific branch, sbommv provides flexibility to handle it efficiently.
