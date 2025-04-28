
# 🔹Github --> S3 Examples 🔹

Fetch SBOM from Github System(adapter) and upload it to S3 System(adapter)

## Overview  

`sbommv` is a tool designed to transfer SBOMs (Software Bill of Materials) between systems. It operates with two different systems. In this case:

- **Input (Source) System** → Fetches SBOMs  --> **Github**
- **Output (Destination) System** → Uploads SBOMs  --> **S3**

### Fetching SBOMs from GitHub Repository

GitHub offers three methods to retrieve/fetch SBOMs from a Repository:

1. **API Method** – Uses GitHub’s [Dependency Graph API](https://docs.github.com/en/enterprise-cloud@latest/rest/dependency-graph/sboms?apiVersion=2022-11-28) to fetch SBOM for a repo.
2. **Release Method** – Extracts SBOMs from Github repository release page.
3. **Tool Method** – Clones the repository and generates SBOMs using Syft  

### Upload SBOMs to S3 bucket

Once SBOMs are fetched, they are uploaded to an S3 bucket. To use S3 as the output adapter, provide:

- **Bucket name** (`--out-s3-bucket-name`)
- Optional **prefix** (`--out-s3-prefix`)
- **Region** (`--out-s3-region`, defaults to us-east-1 if not specified)
- **Credentials** (via `--out-s3-access-key` and `--out-s3-secret-key`, or AWS default config like `~/.aws/` credentials)

The S3 adapter supports two processing modes:

- **Sequential**: Uploads SBOMs one-by-one (default).
- **Parallel**: Uploads up to 3 SBOMs concurrently for faster processing.

Use the --out-s3-processing-mode flag to specify sequential or parallel.

## 1. Basic Transfer(Single Repository): GitHub  → S3

### 1.1 Github Default(Dependency Graph API) Method

```bash
sbommv transfer \
  --input-adapter=github \
  --in-github-url="https://github.com/interlynk-io/sbomqs"\
  --output-adapter=S3 \
  --out-s3-bucket-name="demo-test-sbom" \
  --out-s3-prefix="sbomqs-api" \
  --out-s3-region="us-east-1" \
  --out-s3-access-key="AKIA..." \
  --out-s3-secret-key="wJalr..."
```

OR

Using default credentials, if AWS credentials are configured in `~/.aws/credentials`, omit `--out-s3-access-key`, `--out-s3-secret-key`, and even `--out-s3-region`, if you want to use default region as it is:

```bash
sbommv transfer \
--input-adapter=github \
--in-github-url="https://github.com/interlynk-io/sbomqs" \
--output-adapter=s3 \
--out-s3-bucket-name="demo-test-sbom" \
--out-s3-prefix="sbomqs-api"
```

- **What this does**:
  - Fetches SBOMs from the latest release of the `interlynk-io/sbomqs` repository.
  - Uploads them to `s3://demo-test-sbom/sbomqs-api/`.

**NOTE**:

- Best for repositories that don’t publish SBOMs in releases.

### 1.2 GitHub Release Method

```bash
sbommv transfer \
--input-adapter=github \
--in-github-url="https://github.com/interlynk-io/sbomqs" \
--in-github-method=release \
--output-adapter=s3 \
--out-s3-bucket-name="demo-test-sbom" \
--out-s3-prefix="sbomqs-release" \
--out-s3-region="us-east-1" \
--out-s3-access-key="AKIA..." \
--out-s3-secret-key="wJalr..."
```

- **What this does**:
  - Fetches SBOMs from GitHub’s dependency graph API for the repository `interlynk-io/sbomqs`
  - Uploads them to `s3://demo-test-sbom/sbomqs-release/`.

### 1.3 GitHub Tool Method (SBOM Generation Using Syft)

```bash
sbommv transfer \
--input-adapter=github \
--in-github-url="https://github.com/interlynk-io/sbomqs" \
--in-github-method=tool \
--output-adapter=s3 \
--out-s3-bucket-name="demo-test-sbom" \
--out-s3-prefix="sbomqs-tool" \
--out-s3-region="us-east-1" \
--out-s3-access-key="AKIA..." \
--out-s3-secret-key="wJalr..."
```

- **What this does**:
  - Clones the `interlynk-io/sbomqs` repository.
  - Generates SBOMs using Syft.
  - Uploads them to `s3://demo-test-sbom/sbomqs-tool/`.

**NOTE**:

- Best for repositories without SBOMs in API or releases

### 1.4 Fetch SBOMs for a Specific GitHub Branch (GitHub Tool Method Only)

```bash
sbommv transfer --input-adapter=github --in-github-url="https://github.com/interlynk-io/sbomqs" \
                --in-github-method=tool --in-github-branch="main" \
                --output-adapter=s3 --out-s3-bucket-name="demo-test-sbom" --out-s3-prefix="sbomqs-tool-main" \
                --out-s3-region="us-east-1" --out-s3-access-key="AKIA..." --out-s3-secret-key="wJalr..."
```

- **What this does**:
  - Clones the main branch of `interlynk-io/sbomqs`.
  - Generates SBOMs using Syft.
  - Uploads them to `s3://demo-test-sbom/sbomqs-tool-main/`

**NOTE**:

- Only applicable to the `tool` method because releases and API do not support branches.

## 2. Parallel Upload to S3

```bash
sbommv transfer --input-adapter=github --in-github-url="https://github.com/interlynk-io/sbomqs" \
                --in-github-method=release \
                --output-adapter=s3 --out-s3-bucket-name="demo-test-sbom" --out-s3-prefix="psbomqs" \
                --out-s3-region="us-east-1" --out-s3-access-key="AKIA..." --out-s3-secret-key="wJalr..." \
                --processing-mode="parallel"
```

- **What this does**:
  - Fetches SBOMs from the latest release of `interlynk-io/sbomqs`.
  - Uploads them to `s3://demo-test-sbom/psbomqs/` in parallel (up to 3 concurrent uploads).

**NOTE**:

- Use `--processing-mode="parallel"` for faster uploads with multiple SBOMs.

## 3. Dry-Run Mode (Simulation Only)

```bash
sbommv transfer --input-adapter=github --in-github-url="https://github.com/interlynk-io/sbomqs" \
                --in-github-method=release \
                --output-adapter=s3 --out-s3-bucket-name="demo-test-sbom" --out-s3-prefix="dsbomqs" \
                --out-s3-region="us-east-1" --out-s3-access-key="AKIA..." --out-s3-secret-key="wJalr..." \
                --dry-run
```

**What this does**:

- Simulates fetching SBOMs from `interlynk-io/sbomqs` releases.
- Lists SBOMs that would be uploaded to `s3://demo-test-sbom/dsbomqs/` without performing the upload.

**NOTE**:

- Useful for previewing the transfer process.

## 4. Advanced Transfer(Multiple Repositories in an Organization)

### 4.1 Include Repos of an Organization

#### 4.1.1 Github Release Method

```bash
sbommv transfer --input-adapter=github --in-github-url="https://github.com/sigstore" \
                --in-github-method=release --in-github-include-repos=cosign,rekor \
                --output-adapter=s3 --out-s3-bucket-name="demo-test-sbom" --out-s3-prefix="sigstoreicr" \
                --out-s3-region="us-east-1" --out-s3-access-key="AKIA..." --out-s3-secret-key="wJalr..."
```

**What this does**:

- Fetches SBOMs from the `cosign` and `rekor` repositories under the `sigstore` organization.
- Uploads them to `s3://demo-test-sbom/demo-test-sbom/sigstoreicr/`

Similarly, you can do for **GitHub API Method** and **GitHub Tool Method**.

### 4.2 Exclude Certain Repositories

#### 4.2.1 GitHub Release Method

```bash
sbommv transfer --input-adapter=github --in-github-url="https://github.com/sigstore" \
                --in-github-method=release --in-github-exclude-repos=docs \
                --output-adapter=s3 --out-s3-bucket-name="demo-test-sbom" --out-s3-prefix="sigstore" \
                --out-s3-region="us-east-1" --out-s3-access-key="AKIA..." --out-s3-secret-key="wJalr..."
```

**What this does:**

- Fetches SBOMs from all `sigstore` repositories except docs via releases.
- Uploads them to `s3://demo-test-sbom/esigstore/`

Similarly, you can do for **GitHub API Method** and **GitHub Tool Method**.
