# sbommv

sbommv - Your primary tool to transfer SBOM's between different systems.

## Summary

sbommv is designed to allow transfer sboms across systems. The tool supports input, translation, enrichment & output adapters which allow it to be extensible in the future. Input adapters are responsbile to interface with services and provide various methods to extract sboms. The output adapters handles all the complexity related to uploading sboms. 

**Examples**

1. **Generate & Transfer SBOM's for all repositories in Github org of Interlynk to interlynk SBOM' platform**:
   Generate & Transfer SBOM's from all repositories in the interlynk github organization using github apis, and transfer them to interlynk. If interlynk platform does not contain projects it will create them.

   ```bash
   export GITHUB_TOKEN=ghp_klgJBxKukyaoWA******
   export INTERLYNK_SECURITY_TOKEN=lynk_api******
   sbommv transfer --input-adapter=github --in-github-url="https://github.com/interlynk-io" --output-adapter=interlynk --out-interlynk-url="http://localhost:3000/lynkapi"
   ```

2. **DRY RUN: Transfer SBOM from the latest version of sbomqs to interlynk platform**:
  This will look for the latest release of the repository and check if SBOMs are generated, in dry-run mode, it will just iterate the sboms found, and check if login to the output adapter works.

   ```bash
   export GITHUB_TOKEN=ghp_klgJBxKukyaoWA******
   export INTERLYNK_SECURITY_TOKEN=lynk_api******
   sbommv transfer --input-adapter=github  --in-github-url=https://github.com/interlynk-io/sbomqs --in-github-method="release" --output-adapter=interlynk  --out-interlynk-url=https://api.interlynk.io/lynkapi --dry-run
   ```


## Data Flow 
```
+---------------------+     +------------------------------+     +----------------------+
|    Input Adapter    | --> |    Enrichment/Translation    | --> |   Output Adapter     |
|-------------------- |     |------------------------------|     |----------------------|
|  - GitHub           |     |  - SBOM Translation*         |     |  - Interlynk         |
|  - BitBucket*       |     |  - Enrichment*               |     |  - Dependency-Track* |
|  - Dependency-Track*|     +------------------------------+     |  - Folder*           |
|  - Folder*          |                                          |                      |
+---------------------+                                          +-----------------------+

* Coming Soon
```

## Adapters 


### Input Adapters

#### GitHub

The **GitHub adapter** allows you to extract/download SBOMs from GitHub. The adapter provides the following methods of extracting SBOMs:

- **Release**:  
  This method looks at the releases for the repository and extracts all the SBOMs that follow the recognized file patterns as described by **CycloneDX** & **SPDX** specs.

- **API** *(Default)*:  
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

4. **All repositories from `interlynk-io` using the API method**:

   ```bash
   --in-github-url=https://github.com/interlynk-io 
   ```

---


### Output Adapters 

#### Interlynk

The **Interlynk adapter** allows you to upload SBOMs to Interlynk Enterprise Platform. If no repository name is specified, it will auto-create projects & the env on the platform.
To access this platform `INTERLYNK_SECURITY_TOKEN`, will be required. 

---

**Command-line Parameters Supported by the Adapter**

- `--out-interlynk-url` [Optional]: URL for the interlynk service. Defaults to `https://api.interlynk.io/lynkapi`  
- `--out-interlynk-project-name` [Optional]:  Name of the project to upload the SBOM to, this is optional, if not-provided then it auto-creates it. 
- --out-interlynk-project-env` [Optional]: Defaults to the "default" env.   

---

**Usage Examples**

1. **Upload SBOMs to a particular project**:  

   ```bash
   --out-interlynk-project-name=abc
   ```

2. **Upload SBOMs to a particular project and env**:  

   ```bash
   --out-interlynk-project-name=abc
   --out-interlynk-project-env=production
   ```
   
### Enrichment Adapters 
## License 

### Conversion Adapters
## SPDX -> CDX 

## CDX -> SPDX
