# Output Adapter

Adapters are representation of systems. Sbommv fetches SBOMs from one system and push to another system. The destination/output systems are represented by output adapters. Popular examples are Interlynk, Dependency-Track, folder, security tools or any SBOM analysis platform.

In short, it's responsible for pushing SBOMs to destination. Let's discuss output system one by one:

## 1. Dependency-Track(DTrack)

The **Dependency-Track Adapter** allows you to upload SBOMs to Dependency Track platform to a particular project. If no project name is specified, it will auto-create project. To access this platform `DTRACK_API_KEY`, will be required.

- **DTrack Adapter CLI Parameters**

  - `--out-dtrack-url` [Optional]: URL for the interlynk service. Defaults to `https://api.interlynk.io/lynkapi`  
  - `--out-dtrack-project-name` [Optional]:  Name of the project to upload the SBOM to, this is optional, if not-provided then it auto-creates it.
  - `--out-dtrack-project-version` [Optional]:  Version of the project, this is optional, if not-provided then it feeds "latest" value.

- **Usage Examples**

1. **Upload SBOMs to a particular project with a version "latest"**:  

   ```bash
   --out-dtrack-project-name=xyz
   ```

2. **Upload SBOMs to a particular project with a version "v0.1.0"**:  

   ```bash
   --out-dtrack-project-name=xyz
   --out-dtrack-project-version=v0.1.0
   ```

## 2. Interlynk

The **Interlynk adapter** allows you to upload SBOMs to Interlynk Enterprise Platform. If no repository name is specified, it will auto-create projects & the env on the platform.
To access this platform `INTERLYNK_SECURITY_TOKEN`, will be required.

- **Interlynk Adapter CLI Parameters**

  - `--out-interlynk-url` [Optional]: URL for the interlynk service. Defaults to `https://api.interlynk.io/lynkapi`  
  - `--out-interlynk-project-name` [Optional]:  Name of the project to upload the SBOM to, this is optional, if not-provided then it auto-creates it.
  - `--out-interlynk-project-env` [Optional]: Defaults to the "default" env.

- **Usage Examples**

1. **Upload SBOMs to a particular project**:  

   ```bash
   --out-interlynk-project-name=abc
   ```

2. **Upload SBOMs to a particular project and env**:  

   ```bash
   --out-interlynk-project-name=abc
   --out-interlynk-project-env=production
   ```

## 3. Folder

The **Folder Adapter** allows you to save SBOMs to local Folder. The adapter job is to save SBOMs (Software Bills of Materials) to a local filesystem directory.  It’s designed to save a specified folder. Unlike the Interlynk adapter, which interacts with a remote service, the Folder adapter works with local folders.

- **Folder Adapter specific CLI parameters**

  - `--out-folder-path`: folder path to save SBOMs.  
  - `--out-folder-processing-mode`: Mode of saving SBOMs i.e in sequential/parallel. By default, it's `sequential`.

- **Folder Adapter Usage Examples**

1. **To save SBOM to root folder `temp` in a sequential manner**.
   This will look for the latest release of the repository and check if SBOMs are generated.

   ```bash
   --out-folder-path=temp
   --out-folder-processing-mode="sequential"
   ```

2. **To save SBOMs to root folder `temp` in a parallel/concurrent mode**.
   This will look for the latest release of the repository and check if SBOMs are generated.

   ```bash
   --out-folder-path=temp
   --out-folder-processing-mode="parallel"
