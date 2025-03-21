# ⚙️ sbommv Command Flags & Usage Guide

This guide explains the **CLI flags** available in `sbommv transfer`, detailing how to configure both input and output systems using adapters. It also includes usage guidance to help you apply flags effectively in real-world scenarios.

---

## 🟢 Global Flags

These flags apply to all transfer commands, regardless of the adapter used:

- `--processing-mode`  
  Sets how SBOMs are fetched and uploaded: `"sequential"` *(default)* or `"parallel"`. Parallel mode improves performance on large sets.

- `--dry-run`  
  Simulates a full SBOM transfer (input + output) **without actual uploads**, providing a preview of what will be fetched and where it would be sent.

- `--debug`, `-D`  
  Enables debug logging for detailed execution output.

- `--help`, `-h`  
  Displays the help menu for the current command.

---

## 🔄 Input Adapters

### 1. GitHub Input Adapter

Fetches SBOMs from GitHub repositories or organizations.

#### Required Flag

- `--input-adapter=github`  

#### Adapter-Specific Flags

- `--in-github-url=<URL>`  
  GitHub repository or organization URL.

- `--in-github-version`
  GitHub release version to fetch SBOMs from.
  - "latest" (default) – fetches SBOM from the most recent release.
  - "*" – fetches SBOMs from all available releases.
  - Github API Method is not applicable.
- **NOTE**: On fetching from multiple version, github has request limiter, to avoid it you need to export `GITHUB_TOKEN`

- `--in-github-method=<method>`  
  Method of fetching: `api` *(default)*, `release`, or `tool`.

- `--in-github-branch=<branch>`  
  *(Tool method only)* Branch to scan (e.g., `main`, `develop`).

- `--in-github-include-repos=<repos>`
  *(Org-level only)* Comma-separated list of repos to include.

- `--in-github-exclude-repos=<repos>`  
  *(Org-level only)* Comma-separated list of repos to exclude. Cannot be combined with `--include-repos`.

#### ✅ When to Use These

- Fetch SBOMs from a specific repo for latest version → `--in-github-url=https://github.com/org/repo`  
- Fetch SBOMs from a specific repo for it's all version → `--in-github-url=https://github.com/org/repo`  + `--in-github-version="*"`
- Fetch from all repos in an org → Use org URL + include/exclude filters  
- Scan a specific branch (tool ) → Add `--in-github-branch=main`

---

### 2. Folder Input Adapter

Fetches SBOMs from local folders.

#### Required Flag

- `--input-adapter=folder`

#### Adapter-Specific Flags

- `--in-folder-path=<path>`method  
  Path to the folder to scan.

- `--in-folder-recursive=true|false`  
  Whether to scan subdirectories. Default is `false`.

---

## 📤 Output Adapters

### 3. Dependency-Track Output Adapter

Uploads SBOMs to a Dependency-Track instance. If the specified project doesn't exist, sbommv will auto-create one using the SBOM’s metadata (e.g., name and version). Authentication is handled via the DTRACK_API_KEY environment variable.

#### Required Flag

- `--output-adapter=dtrack`

#### Adapter-Specific Flags

- `--out-dtrack-url=<URL>`
URL of the Dependency-Track API (e.g., <http://localhost:8081>).

- `--out-dtrack-project-name=<name>`
Name of the project in Dependency-Track to upload SBOMs to.
If not provided, sbommv automatically generates the project name based on the source:
- From folder → Derived from the SBOM's primary component name and version.
- From GitHub → Formatted as organization/repo with version set to "latest" by default.

- `--out-dtrack-project-version=<version>`
Version of the project. Defaults to "latest" if not specified.

**NOTE**:

- Make sure to generate `DTRACK_API_KEY` to access Dependency-Track platform.


✅ When to Use These

- Upload SBOMs to a named project → `--out-dtrack-project-name="my-app"`
- Set a specific version → `--out-dtrack-project-version="v1.2.3"`, if not provided, `"latest"` taken as *default*.
- Connect to a self-hosted DTrack instance → `--out-dtrack-url=http://your-dtrack-instance:8080`

### 2. Interlynk Output Adapter

Uploads SBOMs to the **Interlynk SBOM Management Platform**.

#### Required Flag

- `--output-adapter=interlynk`

#### Adapter-Specific Flags

- `--out-interlynk-url=<URL>`
  Interlynk API URL. Defaults to `https://api.interlynk.io/lynkapi`.

- `--out-interlynk-project-name=<name>`  
  Project name to upload SBOMs to. If not provided, it's auto-generated.

- `--out-interlynk-project-env=<env>`  
  Environment to associate with the project. Default is `"default"`.

#### ✅ When to Use These

- Upload to a known project → Provide `--out-interlynk-project-name`  
- Assign an environment (e.g., staging, production) → Use `--out-interlynk-project-env`  

**Note:**

- If project details aren't specified, sbommv will auto-create them.
- Make sure to generate `INTERLYNK_SECURITY_TOKEN` to access Interlynk platform.

---

### 3. Folder Output Adapter

Saves SBOMs locally to a folder.

#### Required Flag

- `--output-adapter=folder`

#### Adapter-Specific Flags

- `--out-folder-path=<path>`  
  Target directory for storing SBOMs.

---

## 📌 **Tips & References**

✅ **Use `--dry-run`** to preview the SBOMs that will be fetched and where they’ll be uploaded—without making changes.

📘 **Examples** → [View Example Commands](https://github.com/interlynk-io/sbommv/blob/main/docs/examples.md)  
🚀 **Getting Started Guide** → [Start Here](https://github.com/interlynk-io/sbommv/blob/main/docs/getting_started.md)
