
# ⚙️ SBOMMV Command Flags & Usage Guide

This guide provides a **detailed breakdown** of the available flags for `sbommv transfer`, explaining their purpose and how to use them effectively.

## 🟢 Global Flags

These flags apply to **all commands**, regardless of input or output adapters.

- `-D, --debug` : Enables **debug logging** to provide detailed insights into execution. Useful for troubleshooting.
- `--dry-run` : **Simulates the transfer process** without actually uploading SBOMs. This allows users to preview the results before making real changes.
- `-h, --help` : Displays the help menu for the `sbommv transfer` command.

## 1. Input Adapter GitHub Flags

The **GitHub adapter** allows fetching SBOMs from repositories or organizations. The following flags control how SBOMs are retrieved.

- `--input-adapter=github`: **Specifies GitHub as the input system** to fetch SBOMs. **(Required for GitHub input)** 
- `--in-github-url <URL>`: **GitHub repository or organization URL** from which to fetch SBOMs. **Supports both individual repositories & organizations.**
- `--in-github-method <method>`: Specifies the method for retrieving SBOMs. **Options:** `"release"`, `"api"`, `"tool"`. Default: `"api"`.
- `--in-github-branch <branch>`: **(Tool Method Only)** Specifies a branch when using the **`tool` method** (e.g., `"main"`, `"develop"`). **Ignored for API and Release methods.**
- `--in-github-include-repos <repos>`: **(Org-Level Fetching Only)** Comma-separated list of repositories to **include** when fetching SBOMs from a GitHub organization. Not used for single repos.
- `--in-github-exclude-repos <repos>`: **(Org-Level Fetching Only)** Comma-separated list of repositories to **exclude** when fetching SBOMs from a GitHub organization. **Cannot be used with `--in-github-include-repos` simultaneously.**

### 📌 When to use these Github Input Adalpter flags?

- **For a specific repository** → Use `--in-github-url="<repo_url>"`
  - Example: `--in-github-url=https://github.com/sigstore/cosign`
- **For an entire organization** → Use `--in-github-url="<org_url>"` with  `--in-github-include-repos` OR `--in-github-exclude-repos`
  - Example: `--in-github-url=https://github.com/sigstore`
- **For a specific branch** → Use `--in-github-branch="<branch_name>"` (only with the tool method)  

## 2. Input Adapter Folder Flags

The **Folder adapter** allows fetching/scanning SBOMs from directories and sub-directories. The following flags control how SBOMs are retrieved.

- `--input-adapter=folder`: **Specifies Folder as the input system** to fetch SBOMs. **(Required)** 
- `--in-folder-path=<path>`: **Folder path** from which to scan/fetch SBOMs.
- `--in-folder-recursive=true`: Specifies to scan from all sub-directories under provided path.
- `--in-folder-processing-mode="parallel`: **SBOM fetching mode**, fetch or scan sboms cuncurrently or parralelly, which is quite faster than sequential mode.


## 1. Output Adapter: Interlynk Flags

The **Interlynk adapter** is used for uploading SBOMs to **Interlynk’s SBOM Management Platform**.

- `--output-adapter=interlynk`: **Specifies Interlynk as the output system.** **(Required for Interlynk output)**
- `--out-interlynk-url <URL>`: **Interlynk API URL**. Defaults to `"https://api.interlynk.io/lynkapi"`. Can be overridden for self-hosted instances.
- `--out-interlynk-project-name <name>`: **Name of the Interlynk project** to upload SBOMs to. If not provided, a project is auto-created based on the repository name.
- `--out-interlynk-project-env <env>`: **Project environment in Interlynk.** Default: `"default"`. Other options: `"development"`, `"production"`.

### 📌 When to use these Interlynk Output Adapter flags?

- **Uploading to a specific Interlynk project** → Use `--out-interlynk-project-name="<name>"`  
- **Uploading to a custom environment** → Use `--out-interlynk-project-env="development"`
- **NOTE**: For entire organization, no need to provide specific project, it will be automatically created as per requirements.

## 2. Ouput Adapter Folder Flags

The **Folder adapter** allows fetching/scanning SBOMs from directories and sub-directories. The following flags control how SBOMs are retrieved.

- `--output-adapter=folder`: **Specifies Folder as the output system** to save SBOMs. **(Required)**
- `--out-folder-path=<path>`: **Folder path** to which to save or download the SBOMs.
- `--out-folder-processing-mode="parallel`: **SBOM saving mode**, save sboms cuncurrently or parralelly, which is quite faster than sequential mode.

## 📌 NOTE

✅ To see the display of how many SBOMs are fetched or how many SBOMs to be uploaded, use `--dry-run`. It provides you rough idea about what being fetched and what would be uploaded.  

📌 **Looking for command examples?** → [Check out the Example Guide](https://github.com/interlynk-io/sbommv/blob/main/docs/examples.md)  
📌 **Want to get started quickly?** → [Read Getting Started with sbommv]([./docs/GettingStarted.md](https://github.com/interlynk-io/sbommv/blob/main/docs/getting_started.md))  
