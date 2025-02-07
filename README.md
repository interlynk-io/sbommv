# sbommv

sbommv - Your primary tool to transfer SBOM's between different systems.

## Summary

sbommv is designed to allow transfer sboms across systems. The tool supports input, translation, enrichment & output adapters which allow it to be extensible in the future. Input adapters are responsbile to interface with services and provide various methods to extract sboms. The output adapters handles all the complexity related to uploading sboms. 

**Examples**

1. **Transfer SBOM from the latest version of sbomqs to interlynk platform**:
  This will look for the latest release of the repository and check if SBOMs are generated, if found it will create a new project with the repo-name in interlynk and upload it. 

   ```bash
   export GITHUB_TOKEN=ghp_klgJBxKukyaoWA******
   export INTERLYNK_SECURITY_TOKEN=lynk_api******
   sbommv transfer --in-github-url=https://github.com/interlynk-io/sbomqs --out-interlynk-url=https://api.interlynk.io/lynkapi
   ```

2. **DRY RUN: Transfer SBOM from the latest version of sbomqs to interlynk platform**:
  This will look for the latest release of the repository and check if SBOMs are generated, in dry-run mode, it will just iterate the sboms found, and check if login to the output adapter works.

   ```bash
   export GITHUB_TOKEN=ghp_klgJBxKukyaoWA******
   export INTERLYNK_SECURITY_TOKEN=lynk_api******
   sbommv transfer --in-github-url=https://github.com/interlynk-io/sbomqs --out-interlynk-url=https://api.interlynk.io/lynkapi --dry-run
   ```


## Data Flow 
```
+---------------------+     +------------------------------+     +----------------------+
|    Input Adapter    | --> |    Enrichment/Translation    | --> |   Output Adapter     |
|-------------------- |     |------------------------------|     |----------------------|
|  - GitHub           |     |  - SBOM Translation*         |     |  - Interlynk         |
|  - BitBucket*       |     |  - Enrichment*               |     |  - Dependency-Track* |
|  - Dependency-Track*|     +------------------------------+     |                      |
+---------------------+                                          +-----------------------+

* Coming Soon
```

## Adapters 


### Input Adapters

#### GitHub

The **GitHub adapter** allows you to extract/download SBOMs from GitHub. The adapter provides the following methods of extracting SBOMs:

- **Release** *(Default)*:  
  This method looks at the releases for the repository and extracts all the SBOMs that follow the recognized file patterns as described by **CycloneDX** & **SPDX** specs.

- **API**:  
  This method uses the GitHub API to download **SPDX** SBOM for the repository, if available.

- **Tool**:  
  This method clones the repository and runs your tool of choice to generate the SBOM.

---

**Command-line Parameters Supported by the Adapter**

- `--in-github-url`: Takes the repository or owner URL for GitHub.  
- `--in-github-include-repos`: Specifies repositories from which SBOMs should be extracted.  
- `--in-github-exclude-repos`: Specifies repositories to exclude from SBOM extraction.  
- `--in-github-method`: Specifies the method of extraction (`release`, `api`, or `tool`).  

---

**Usage Examples**

1. **For the latest release version of `sbomqs` using the release method**:  
   This will look for the latest release of the repository and check if SBOMs are generated.

   ```bash
   --in-github-url=https://github.com/interlynk-io/sbomqs
   ```

2. **For a particular release (`v1.0.0`) of `sbomqs` using the release method**:

   ```bash
   --in-github-url=https://github.com/interlynk-io/sbomqs@v1.0.0
   ```

3. **For only certain repositories (`sbomqs`, `sbomasm`) of `interlynk-io` using the API method**:

   ```bash
   --in-github-url=https://github.com/interlynk-io \
   --in-github-include-repos=sbomqs,sbomasm \
   --in-github-method=api
   ```

4. **To exclude specific repositories (`sbomqs`) from `interlynk-io` using the API method**:

   ```bash
   --in-github-url=https://github.com/interlynk-io \
   --in-github-exclude-repos=sbomqs \
   --in-github-method=api
   ```

---






### Output Adapters 
#### Interlynk


### Enrichment Adapters 

### Conversion Adapters
