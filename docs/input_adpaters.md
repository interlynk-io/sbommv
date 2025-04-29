# Input Adapters and Respective Flags

In sbommv, **input adapters** are responsible for fetching SBOMs from supported source systems. These sources include:

- GitHub (via API, releases, or external SBOM tools),
- Local folders,
- AWS S3 bucket,
- Interlynk platform *(upcoming)*,
- Dependency-Track *(upcoming)*.

Each input adapter exposes **CLI flags** specific to its configuration and behavior. This guide outlines the available input adapters, how they work, and the flags needed to use them.

## 1. GitHub Adapter

Fetches SBOMs from GitHub repositories. Supports three methods:

- **API (default)** – Uses GitHub’s Dependency Graph API to fetch an SPDX-JSON SBOM for the default branch.  
- **Release** – Downloads SBOM artifacts from the repository’s Releases section.  
- **Tool** – Clones the repo and generates SBOMs using tools like `syft`.

- **Supported Flags**

- `--in-github-url` – Repository or organization URL.  
- `--in-github-method` – Extraction method: `api`, `release`, or `tool`.  
- `--in-github-version` – (Optional) Specific release tag (e.g., `v1.0.0`).  
- `--in-github-include-repos` – Comma-separated list of repos to include.  
- `--in-github-exclude-repos` – Comma-separated list of repos to exclude.

- **Usage Examples**

```bash
# Fetch from latest release
--in-github-url=https://github.com/interlynk-io/sbomqs
--in-github-method="release"

# Fetch from a specific release tag
--in-github-version="v1.0.0"

# Include specific repos from an org
--in-github-url=https://github.com/interlynk-io
--in-github-include-repos=sbomqs,sbomasm

# Exclude specific repos from an org
--in-github-exclude-repos=sbomqs
```

---

## 2. Folder Adapter

Scans a local folder for valid SBOM files.

- **Supported Flags**

- `--in-folder-path` – Path to the root folder.  
- `--in-folder-recursive` – `true` or `false`. Defaults to `false`.  

- **Usage Examples**

```bash
# non-recursive
--in-folder-path=sboms_ws
--in-folder-recursive=false

--in-folder-recursive=true
```

---

## 3. AWS S3 Adapter

Fetch SBOMs from S3 buckets using object paths or filters.

- **S3 Supported Flags**


- `--in-s3-bucket-name=<bucket_name>`  – Bucket Name.

- `--in-s3-prefix=<prefix_name>`  – Prefix Name, similar of sub-folder name.

- `--in-s3-access-key=<AWS ACCESS KEY>` – AWS Access Key or aws credentials already present at `~/.aws`

- `--in-s3-secret-key=<AWS SECRET KEY` – AWS Secret Key or aws credentials already present at `~/.aws`

- `--in-s3-region=<region>` – If not provided or empty, then `us-east-1` is taken as default value.

- **Usage Examples**

```bash
# input adapter S3
--input-adapter=s3 

# with a bucket name "demo-test-sbom"
--in-s3-bucket-name="demo-test-sbom" 

# with a prefix name "dropwizard"
--in-s3-prefix="dropwizard" 

# with a region "us-east-1" as by-default
--in-s3-region="" 

# prvided AWS access key
--in-s3-access-key=$AWS_ACCESS_KEY 

# prvided AWS secret key
--in-s3-secret-key=$AWS_SECRET_KEY
```

---

## Coming Soon

- **AWS S3 Adapter** – Fetch SBOMs from S3 buckets using object paths or filters.  
- **Interlynk Adapter** – Pull SBOMs using project ID from the Interlynk platform.  
- **Dependency-Track Adapter** – Fetch SBOMs by project UUID from Dependency-Track.

---

## Summary

Input adapters allow sbommv to ingest SBOMs from multiple sources in a standardized way. This document outlines the CLI flags required to configure each adapter so that they can be dropped into automation workflows with ease.
