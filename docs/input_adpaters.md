
# Input Adapter

Adapters are representation of systems. Sbommv fetches SBOMs from one system and push to another system. The input systems are represented by input adapters. Popular examples of input adapters are e.g., GitHub, AWS, folders, local files or any system with source of SBOMs.

In short, it's responsible for fetching SBOMs from sources. Let's discuss input system one by one:

## 1. GitHub

The **GitHub adapter** allows you to extract/download SBOMs from GitHub. This adapter provides the following methods of extracting SBOMs:

- **Release**:  
  This method looks at the releases page for the repository and extracts all the SBOMs that follow the recognized file patterns as described by **CycloneDX** & **SPDX** specs.

- **API** *(Default)*:
  This method uses the Dependency Graph API to download **SPDX-JSON** SBOM for the repository, if available.

- **Tool**:  
  This method clones the repository and runs your tool of choice to generate the SBOM.

  **NOTE**: Currently we support [syft](https://github.com/anchore/syft), in future will add support for more sbom generating tools.

- **Github Adapter specific CLI parameters**

  - `--in-github-url`: Takes the repository or owner URL for GitHub.  
  - `--in-github-include-repos`: Specifies repositories from which SBOMs should be extracted.  
  - `--in-github-exclude-repos`: Specifies repositories to exclude from SBOM extraction.  
  - `--in-github-method`: Specifies the method of extraction (`release`, `api`, or `tool`).  

- **Github Adapter Usage Examples**

1. **For the latest release version of `sbomqs` using the release method**:  
   This will look for the latest release of the repository and check if SBOMs are generated.

   ```bash
   --in-github-url=https://github.com/interlynk-io/sbomqs
   --in-github-method="release"
   ```

2. **For a particular release (`v1.0.0`) of `sbomqs` using the release method**:

   ```bash
   --in-github-url=https://github.com/interlynk-io/sbomqs@v1.0.0
   --in-github-method="release"
   ```

3. **For only certain repositories (`sbomqs`, `sbomasm`) of `interlynk-io` using the API method**:

   ```bash
   --in-github-url=https://github.com/interlynk-io \
   --in-github-include-repos=sbomqs,sbomasm 
   ```

4. **To exclude specific repositories (`sbomqs`) from `interlynk-io` using the API method**:

   ```bash
   --in-github-url=https://github.com/interlynk-io \
   --in-github-exclude-repos=sbomqs
   ```

5. **All repositories from `interlynk-io` using the API method**:

   ```bash
   --in-github-url=https://github.com/interlynk-io 
   ```

## 2. Folder

The **Folder Adapter** allows you to extract/fetch SBOMs from local Folder. The adapter job is to fetch SBOMs (Software Bills of Materials) from a local filesystem directory.  It’s designed to scan a specified folder, optionally including subdirectories. Unlike the GitHub adapter, which interacts with a remote service, the Folder adapter works with local files.

- **Folder Adapter specific CLI parameters**

  - `--in-folder-path`: Takes the folder path.  
  - `--in-folder-recursive`: Specifies whether to scan within sub-directories. By default(`false`), it doesn't scn within sub-directories.
  - `in-folder-processing-mode`: Mode of fetching SBOMs, in sequential/parallel. By default, it's `sequential`.

- **Folder Adapter Usage Examples**

1. **To fetch SBOM from root folder `sboms_ws` in a sequential manner**.
   This will look for the latest release of the repository and check if SBOMs are generated.

   ```bash
   --in-folder-path=sboms_ws
   --in-folder-recursive=false
   --in-folder-processing-mode="sequential"
   ```

2. **To fetch SBOM from root folder `sboms_ws` as well as it's sub-directories in a sequential mode**.
   This will look for the latest release of the repository and check if SBOMs are generated.

   ```bash
   --in-folder-path=sboms_ws
   --in-folder-recursive=true
   --in-folder-processing-mode="sequential"
   ```

3. **To fetch SBOM from root folder `sboms_ws` as well as it's sub-directories in a parallel mode**.
   This will look for the latest release of the repository and check if SBOMs are generated.

   ```bash
   --in-folder-path=sboms_ws
   --in-folder-recursive=true
   --in-folder-processing-mode="parallel"

