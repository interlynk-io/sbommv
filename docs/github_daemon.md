# GitHub Daemon Mode for SBOM Monitoring

## Overview

The GitHub Daemon Mode feature in sbommv enables **continuous monitoring of external GitHub repositories for new releases and automatically fetches their Software Bill of Materials (SBOMs) using a specified method (release, api, or tool)**. It’s designed as a lightweight, flexible component of sbommv’s glue layer, bridging GitHub repositories with output platforms like local folders (folder adapter) or DependencyTrack (dtrack adapter). This feature is ideal for users who need to track SBOMs for repositories they don’t own, such as sigstore/cosign or owner/stree, without manual intervention.

## Key capabilities:

**Continuous Polling**: Periodically checks repositories for new releases (default: every 24 hours).

**SBOM Fetching**: Retrieves SBOMs using configurable methods:

- **release**: Fetches SBOMs from release assets.
- **api**: Uses GitHub’s Dependency Graph API.
- **tool**: Generates SBOMs with Syft.

**Cache Management**: Tracks repository states and SBOMs to avoid redundant fetching, pruning old SBOMs for efficiency.

**Asset Delay Handling**: Waits 3 minutes for assets to ensure GitHub Actions/workflows have time to upload SBOMs.

## How It Works

The daemon mode operates by polling GitHub repositories, comparing release information, and fetching SBOMs when new releases are detected. Here’s a step-by-step breakdown:

### 1. Polling Mechanism

**Purpose**: Detects new releases in monitored repositories.
**Process**:

- The daemon runs a polling loop with a configurable interval (default: 24 hours, set via `--in-github-poll-interval`).

- For each repository (e.g., owner/stree), it queries the GitHub API to fetch the latest release’s `release_id` and `published_at`.

- It compares these with cached values in `sbommv/cache.json` under `cache[adapter][github][method][repos][repo]`.

- If `release_id` or `published_at`  differs, a new release is detected, triggering SBOM fetching.

If they match, no new release exists, and polling continues.

### 2. Asset Delay Handling

**Purpose**: Ensures SBOM assets are available, as GitHub Actions/workflows may delay asset uploads after a release is created.

**Process**:

- When a new release is detected, the daemon waits for a configurable delay (default: 3 minutes, set via `--in-github-asset-wait-delay`) before fetching assets.

- This accounts for typical delays in workflows (e.g., building and uploading SBOMs for stree).

- The wait is applied only for the release method, as api and tool methods don’t rely on release assets.

### 3. SBOM Fetching

**Purpose**: Retrieves SBOMs for the new release using the specified method.

**Process**:

- **release** Method: Queries release assets via GitHub API (e.g., stree-darwin-amd64.spdx.sbom).

- **api** Method: Fetches a single SBOM from GitHub’s Dependency Graph API (dependency-graph-sbom.json).

- **tool** Method: Clones the repo at the release’s commit and generates an SBOM using Syft (syft-generated-sbom.json).

Duplicate SBOMs are skipped by checking the cache (`data[adapter][github][method][sboms][sbomCacheKey]`).

### 4. Cache Management

**Purpose**: Tracks repository states and SBOMs to optimize polling and avoid redundant fetching.

**Cache Structure** (sbommv/cache.json):

- **Repos**: Stores the latest release’s published_at and release_id for each repo (e.g., `data[folder][github][release][repos][owner/stree]`).

- **SBOMs**: Tracks processed SBOMs to prevent duplicates (e.g., `data[folder][github][release][sboms][owner/stree:220351508:stree-darwin-amd64.spdx.sbom]`).

Uses a JSON-based key-value structure for simplicity and readability.

**Pruning**: When a new release is detected, the sboms map for the repo and method (e.g., release) is cleared to store only the latest release’s SBOMs.

This prevents cache bloat from accumulating SBOMs for old releases (e.g., 220288414, 220320120).

Pruning is applied only for the release method, as api and tool methods typically yield one SBOM per release.

### 5. Output

**Purpose**: Delivers fetched SBOMs to the configured output adapter.

**Process**: SBOMs are sent to the adapter (e.g., folder saves to disk, dtrack uploads to DependencyTrack).

The cache is updated (Repos and SBOMs) only if SBOMs are found (for release) or for api/tool methods.

## Design Q/A

### Why Polling?

GitHub doesn’t support third-party webhooks for repositories you don’t own, making polling the best approach for external repos like sigstore/cosign or owner/stree.

A 24-hour default interval balances timeliness and API rate limit usage (configurable via --in-github-poll-interval).

### Why JSON Cache?

A JSON-based key-value cache (`sbommv/cache.json`) is lightweight, human-readable, and dependency-free, aligning with sbommv’s “no internal state” design.

Compared to an embedded SQLite cache, JSON is simpler for small caches (~100 KB for stree), avoiding the complexity of SQL queries and CGO dependencies. See JSON vs. SQLite Rationale for details.

The nested structure (adapter:github:method:repos/sboms) supports multiple adapters and methods efficiently.

### Why 3-Minute Wait ?

GitHub Actions/workflows often delay asset uploads (e.g., SBOMs for stree) by seconds to minutes after a release is created.

A 3-minute wait (`--in-github-asset-wait-delay=180`) covers typical delays, ensuring assets are available before fetching.

Configurable to accommodate varying workflow speeds (e.g., 60s for fast workflows, 300s for slow ones).

### Why Prune SBOMs?

Storing SBOMs for all releases (e.g., 220288414, 220351508) bloats the cache, increasing memory usage and cache.json size.

Pruning to the latest release’s SBOMs keeps the cache lean (e.g., 6 SBOMs for stree:220351508), as older SBOMs are rarely reprocessed.

Applied only for release method, as api/tool yield single SBOMs, naturally limiting cache growth.
