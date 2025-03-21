# Input Adapters and Respective Flags

In sbommv, **input adapters** are responsible for fetching SBOMs from supported source systems. These sources include:

- GitHub (via API, releases, or external SBOM tools),
- Local folders,
- AWS S3 buckets *(upcoming)*,
- Interlynk platform *(upcoming)*,
- Dependency-Track *(upcoming)*.

Each input adapter exposes **CLI flags** specific to its configuration and behavior. This guide outlines the available input adapters, how they work, and the flags needed to use them.

## 1. GitHub Adapter

Fetches SBOMs from GitHub repositories. Supports three methods:

- **API (default)** – Uses GitHub’s Dependency Graph API to fetch an SPDX-JSON SBOM for the default branch.  
- **Release** – Downloads SBOM artifacts from the repository’s Releases section.  
- **Tool** – Clones the repo and generates SBOMs using tools like `syft`.

### **Supported Flags**

- `--in-github-url` – Repository or organization URL.  
- `--in-github-method` – Extraction method: `api`, `release`, or `tool`.  
- `--in-github-version` – (Optional) Specific release tag (e.g., `v1.0.0`).  
- `--in-github-include-repos` – Comma-separated list of repos to include.  
- `--in-github-exclude-repos` – Comma-separated list of repos to exclude.

### **Usage Examples**

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

### Supported Flags

- `--in-folder-path` – Path to the root folder.  
- `--in-folder-recursive` – `true` or `false`. Defaults to `false`.  

### Usage Examples

```bash
# non-recursive
--in-folder-path=sboms_ws
--in-folder-recursive=false

--in-folder-recursive=true
```

---

## Coming Soon

- **AWS S3 Adapter** – Fetch SBOMs from S3 buckets using object paths or filters.  
- **Interlynk Adapter** – Pull SBOMs using project ID from the Interlynk platform.  
- **Dependency-Track Adapter** – Fetch SBOMs by project UUID from Dependency-Track.

---

## Summary

Input adapters allow sbommv to ingest SBOMs from multiple sources in a standardized way. This document outlines the CLI flags required to configure each adapter so that they can be dropped into automation workflows with ease.
