# GitHub Daemon Mode for SBOM Monitoring

## Overview

The GitHub Daemon Mode feature in sbommv enables **continuous monitoring of external GitHub repositories for new releases and automatically fetches their Software Bill of Materials (SBOMs) using a specified method (release, api, or tool)**. It’s designed as a lightweight, flexible component of sbommv’s glue layer, bridging GitHub repositories with output platforms like DependencyTrack (dtrack adapter), Interlynk (interlynk adapter), AWS S3(s3 adapter) and local folders (folder adapter). This feature is ideal for users who need to track SBOMs for repositories they don’t own, such as `interlynk-io/cosign` or any, without manual intervention.

## Key capabilities

**Continuous Polling**: Periodically checks repositories for new releases (default: every 24 hours).

**SBOM Fetching**
Retrieves SBOMs using specified methods:

- **release**: Fetches SBOMs from release assets.
- **api**: Uses GitHub’s Dependency Graph API.
- **tool**: Generates SBOMs with Syft.

**Cache Management**
Tracks repository states and SBOMs to avoid redundant fetching.

**Asset Delay Handling**
Waits 3 minutes for assets to ensure GitHub Actions/workflows have time to upload SBOMs.

## How It Works

The daemon mode operates by polling GitHub repositories, comparing release information(`released_id` and `published_at`), and fetching SBOMs when new releases are detected. Here’s a step-by-step breakdown:

### 1. Polling Mechanism

**Purpose**: Detects new releases in monitored repositories.
**Process**:

- The daemon runs a polling loop with a configurable interval (default: 24 hours, set via `--in-github-poll-interval`).

- For each repository (for example, `interlynk-io/sbomqs` repo), it queries the GitHub API to fetch the latest release’s `release_id` and `published_at`.

- It compares these with cached values stored in sqlite embedded db `sbommv/cache_<output_adapter>_<github_method>.db`.

- If `release_id` or `published_at`  differs, a new release is detected, triggering SBOM fetching.

If they match, no new release exists, and polling continues.

### 2. Asset Delay Handling

**Purpose**: Ensures SBOM assets are available, as GitHub Actions/workflows may delay asset uploads after a release is created.

**Process**:

- When a new release is detected, the daemon waits for a configurable delay (default: 3 minutes, set via `--in-github-asset-wait-delay`) before fetching assets.

- This accounts for typical delays in workflows (e.g., building and uploading SBOMs for repositories).

### 3. SBOM Fetching

**Purpose**: Retrieves SBOMs for the new release using the specified method.

**Process**:

- **release** Method: Queries release assets from GitHub repository release page (e.g., stree-darwin-amd64.spdx.sbom).

- **api** Method: Fetches a single SBOM from GitHub’s Dependency Graph API (dependency-graph-sbom.json).

- **tool** Method: Clones the repo at the release’s commit and generates an SBOM using Syft (syft-generated-sbom.json).

### 4. Cache Management

**Purpose**: Tracks repository states and SBOMs to optimize polling and avoid redundant fetching.  
**Cache Structure**:

- **Database Files**: Uses method-specific SQLite databases (e.g., `.sbommv/cache_<output_adapter>_<github_method>.db`, such as `.sbommv/cache_dtrack_api.db` for `dtrack` with `api` method).

- **Repos Table**: Stores the latest release’s `published_at` and `release_id` for each repo (e.g., `repos` table entry for `interlynk-io/sbomqs`).

- **SBOMs Table**: Tracks processed SBOMs to prevent duplicates (e.g., `sboms` table entry for `interlynk-io/sbomqs:220351508:sbomqs-v0.0.21.spdx.sbom`).

- **Method-Specific Caches**: Each combination of output adapter and GitHub method has its own cache file to prevent overwrites (e.g., `.sbommv/cache_dtrack_release.db`, `.sbommv/cache_dtrack_api.db`).

### 5. Output

**Purpose**: Delivers fetched SBOMs to the configured output adapter.

**Process**:

- SBOMs are sent to the adapter (e.g., `folder` saves to disk, `dtrack` uploads to DependencyTrack, `interlynk` uploads to Interlynk, `s3` uploads to an S3 bucket).
- The cache is updated (`repos` and `sboms` tables) only if SBOMs are found (for `release`) or for `api`/`tool` methods.

## Design Q/A

### Why Polling?

GitHub doesn’t support third-party webhooks for repositories you don’t own, making polling the best approach for external repos like `interlynk-io/sbomqs` or any other repositories.

A 24-hour default interval balances timeliness and API rate limit usage (configurable via `--in-github-poll-interval`).

### Why SQLite Cache?

We initially used a JSON-based key-value cache (`sbommv/cache.json`) for its simplicity, but switched to SQLite databases (e.g., `.sbommv/cache_dtrack_api.db`) for better performance and concurrency handling:

- **Concurrency**: SQLite with WAL mode supports concurrent access by multiple adapter instances (e.g., `dtrack` with `api`, `tool`, `release`), but we later disabled WAL mode since method-specific caches eliminate shared database scenarios.

- **Method-Specific Caches**: Each adapter-method pair (e.g., `dtrack` with `api`) has its own database to prevent overwrites and ensure coherence.

- **Structure**: Uses `repos` and `sboms` tables for efficient querying and updates, replacing the nested JSON structure.

### Why 3-Minute Wait ?

GitHub Actions/workflows often delay asset uploads (e.g., SBOMs for `interlynk-io/sbomqs`) by seconds to minutes after a release is created.

A 3-minute default wait (`--in-github-asset-wait-delay="180s"`) covers typical delays, ensuring assets are available before fetching.

Configurable to accommodate varying workflow speeds (e.g., `60s` for fast workflows, `300s` for slow ones), supporting formats like `60s`, `10m`, `10hr`, or plain seconds.

### Why Prune SBOMs?

Storing SBOMs for all releases (e.g., 220288414, 220351508) bloats the cache, increasing memory usage and cache.json size.

Pruning to the latest release’s SBOMs keeps the cache lean (e.g., 6 SBOMs for stree:220351508), as older SBOMs are rarely reprocessed.

Applied only for release method, as api/tool yield single SBOMs, naturally limiting cache growth.
