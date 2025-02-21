# `sbommv`: Sbom transfers made easy

[![Go Reference](https://pkg.go.dev/badge/github.com/interlynk-io/sbommv.svg)](https://pkg.go.dev/github.com/interlynk-io/sbommv)
[![Go Report Card](https://goreportcard.com/badge/github.com/interlynk-io/sbommv)](https://goreportcard.com/report/github.com/interlynk-io/sbommv)
![GitHub all releases](https://img.shields.io/github/downloads/interlynk-io/sbommv/total)

`sbommv` is your primary tool to transfer SBOM's between different systems.It is designed to allow transfer sboms across systems. The tool supports input, translation, enrichment & output adapters which allow it to be extensible in the future. Input adapters are responsbile to interface with services and provide various methods to extract sboms. The output adapters handles all the complexity related to uploading sboms.

```console
brew tap interlynk-io/interlynk
brew install sbommv
```

Other [installation options](#installation).

# SBOM Platform - [Interlynk](https://app.interlynk.io/)

Our SBOM Automation Platform has a new free tier that provides a comprehensive solution to manage SBOMs (Software Bill of Materials) effortlessly. From centralized SBOM storage, built-in SBOM editor, continuous vulnerability mapping and assessment, and support for organizational policies, all while ensuring compliance and enhancing software supply chain security using integrated SBOM quality scores. The free tier is ideal for small teams.
[Try now](https://app.interlynk.io/)

## Why sbommv

### The Problem: Managing SBOMs Across Systems

A Software Bill of Materials (SBOM) plays a crucial role in software supply chain security, compliance, and vulnerability management. However, organizations face a key challenge: **How do you efficiently move SBOMs between different systems?**.

There are two major categories of systems dealing with SBOMs: **SBOM Sources** (Input/Source Systems) and **SBOM Consumers** (Output/Target Systems). Manually fetching SBOMs from input systems and uploading them to output systems is: Time-consuming, Error-prone and Difficult to scale.

### The Solution: Automating SBOM Transfers(sbommv)

sbommv is designed to move SBOMs between systems effortlessly. It provides: **Input Adapters**(Fetch SBOMs from different sources) and **Output Adapters**(Upload SBOMs to analysis and security platforms). Currently sbommv support following input and output adapters:

- Input Adapters --> github, folder
- Output Adapters --> interlynk, folder

## How sbommv Works ?

- Extract SBOMs from Input Systems (GitHub, local folders, package registries, etc.)
- Transform or Enrich SBOMs (Future capabilities for format conversion, enrichment)
- Send SBOMs to Output Systems(Interlynk, Dependency-Track(coming soon), folders, Security tools, SBOM repositories, compliance platforms). Whereas the folder output system is just for testing purpose, to see the SBOMs you fetched from Input Adapters.

### Examples

#### 1. Pre-requistic:

- Generate **INTERLYNK_SECURITY_TOKEN** with the help of this [resource](https://github.com/interlynk-io/sbommv/blob/main/docs/getting_started.md#2-configuring-interlynk-authentication).
- Until and unless you get an *Rate Limitor error*, github API will work for you. But as you get, generate  GITHUB_TOKEN from [here](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/managing-your-personal-access-tokens#creating-a-personal-access-token-classic).

```bash
   # export the tokens
   export GITHUB_TOKEN=ghp_klgJBxKukyaoWA******
   export INTERLYNK_SECURITY_TOKEN=lynk_api******
```

#### 2. Generate & Transfer SBOM's for all repositories in Github org of `interlynk-io` to Interlynk SBOM's platform

Generate & Transfer SBOM's from all repositories in the `interlynk-io` github organization using github apis, and transfer them to interlynk. If interlynk platform does not contain projects it will create them.

```bash
   sbommv transfer --input-adapter=github --in-github-url="https://github.com/interlynk-io" --output-adapter=interlynk --out-interlynk-url="http://localhost:3000/lynkapi"
```

#### 3. DRY RUN: Transfer SBOM from the latest version of sbomqs to interlynk platform:

This will look for the latest release of the repository and check if SBOMs are generated, in dry-run mode, it will just iterate the sboms found, and check if login to the output adapter works.

```bash
   sbommv transfer --input-adapter=github  --in-github-url=https://github.com/interlynk-io/sbomqs --in-github-method="release" --output-adapter=interlynk  --out-interlynk-url=https://api.interlynk.io/lynkapi --dry-run
```

## What's next 🚀 ??

- [Getting started](https://github.com/interlynk-io/sbommv/blob/main/docs/getting_started.md) with sbommv.
- Try out more [examples](https://github.com/interlynk-io/sbommv/blob/main/docs/examples.md)
- Detailed CLI command and it's flag [usage](https://github.com/interlynk-io/sbommv/blob/main/docs/flag_usage.md)
- More about [Input and Output adapters](https://github.com/interlynk-io/sbommv/blob/main/docs/adapters.md)

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

At its core, sbommv acts as a bridge, seamlessly connecting SBOM source systems (e.g., GitHub, AWS, folders, local files) with SBOM consumer systems (e.g., Interlynk, Dependency-Track, folder, security tools) — eliminating manual work.
To achieve this, sbommv follows an **adapter-based architecture**, where different systems are abstracted as input and output adapters

### Input Adapters

Responsible for fetching SBOMs from various sources.

#### 1. GitHub

The **GitHub adapter** allows you to extract/download SBOMs from GitHub. The adapter provides the following methods of extracting SBOMs:

- **Release**:  
  This method looks at the releases for the repository and extracts all the SBOMs that follow the recognized file patterns as described by **CycloneDX** & **SPDX** specs.

- **API** *(Default)*:  
  This method uses the GitHub API to download **SPDX** SBOM for the repository, if available.

- **Tool**:  
  This method clones the repository and runs your tool of choice to generate the SBOM.

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

#### 2. Folder Intput Adapter

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

### Output Adapters

Responsible for uploading SBOMs to SBOM platforms, Security Platforms, etc.

#### 1. Interlynk

The **Interlynk adapter** allows you to upload SBOMs to Interlynk Enterprise Platform. If no repository name is specified, it will auto-create projects & the env on the platform.
To access this platform `INTERLYNK_SECURITY_TOKEN`, will be required.

- **Interlynk Adapter CLI Parameters**

  - `--out-interlynk-url` [Optional]: URL for the interlynk service. Defaults to `https://api.interlynk.io/lynkapi`  
  - `--out-interlynk-project-name` [Optional]:  Name of the project to upload the SBOM to, this is optional, if not-provided then it auto-creates it.
  - --out-interlynk-project-env` [Optional]: Defaults to the "default" env.

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

#### 2. Folder Output Adapter

The **Folder Adapter** allows you to save SBOMs to local Folder. The adapter job is to save SBOMs (Software Bills of Materials) to a local filesystem directory.  It’s designed to save a specified folder. Unlike the Interlynk adapter, which interacts with a remote service, the Folder adapter works with local folders.

- **Folder Adapter specific CLI parameters**

  - `--out-folder-path`: folder path to save SBOMs.  
  - `--in-folder-recursive`: Specifies whether to scan within sub-directories. By default(`false`), it doesn't scn within sub-directories.
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

# Installation

## Using Prebuilt binaries

```console
https://github.com/interlynk-io/sbommv/releases
```

## Using Homebrew

```console
brew tap interlynk-io/interlynk
brew install sbommv
```

## Using Go install

```console
go install github.com/interlynk-io/sbommv@latest
```

## Using repo

This approach involves cloning the repo and building it.

1. Clone the repo `git clone git@github.com:interlynk-io/sbommv.git`
2. `cd` into `sbommv` folder
3. make; make build
4. To test if the build was successful run the following command `./build/sbommv version`

# Contributions

We look forward to your contributions, below are a few guidelines on how to submit them

- Fork the repo
- Create your feature/bug branch (`git checkout -b feature/bug`)
- Commit your changes (`git commit -aSm "awesome new feature"`) - commits must be signed
- Push your changes (`git push origin feature/new-feature`)
- Create a new pull-request

# Other Open Source Software tools for SBOMs

- [SBOM Quality Score](https://github.com/interlynk-io/sbomqs) - Quality & Compliance tool
- [SBOM Assembler](https://github.com/interlynk-io/sbomasm) - A tool to compose a single SBOM by combining other SBOMs or parts of them
- [SBOM Quality Score](https://github.com/interlynk-io/sbomqs) - A tool for evaluating the quality and completeness of SBOMs
- [SBOM Search Tool](https://github.com/interlynk-io/sbomagr) - A tool to grep style semantic search in SBOMs
- [SBOM Explorer](https://github.com/interlynk-io/sbomex) - A tool for discovering and downloading SBOMs from a public repository

# Contact

We appreciate all feedback. The best ways to get in touch with us:

- ❓& 🅰️ [Slack](https://join.slack.com/t/sbomqa/shared_invite/zt-2jzq1ttgy-4IGzOYBEtHwJdMyYj~BACA)
- :phone: [Live Chat](https://www.interlynk.io/#hs-chat-open)
- 📫 [Email Us](mailto:hello@interlynk.io)
- 🐛 [Report a bug or enhancement](https://github.com/interlynk-io/sbomex/issues)
- :x: [Follow us on X](https://twitter.com/InterlynkIo)

# Stargazers

If you like this project, please support us by starring it.

[![Stargazers](https://starchart.cc/interlynk-io/sbommv.svg)](https://starchart.cc/interlynk-io/sbommv)
