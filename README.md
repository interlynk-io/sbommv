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

# SBOM Platform - Free Tier

Our SBOM Automation Platform has a new free tier that provides a comprehensive solution to manage SBOMs (Software Bill of Materials) effortlessly. From centralized SBOM storage, built-in SBOM editor, continuous vulnerability mapping and assessment, and support for organizational policies, all while ensuring compliance and enhancing software supply chain security using integrated SBOM quality scores. The free tier is ideal for small teams. [Sign up](https://app.interlynk.io/)


## Why sbommv

### The Problem: Managing SBOMs Across Systems

A Software Bill of Materials (SBOM) plays a crucial role in software supply chain security, compliance, and vulnerability management. However, organizations face a key challenge: **How do you efficiently move SBOMs between different systems?**.

There are two major categories of systems dealing with SBOMs: SBOM Sources (Input Systems) and SBOM Consumers (Output Systems). Manually fetching SBOMs from input systems and uploading them to output systems is: Time-consuming, Error-prone and Difficult to scale.

### The Solution: Automating SBOM Transfers(sbommv)

sbommv is designed to move SBOMs between systems effortlessly. It provides: **Input Adapters**(Fetch SBOMs from different sources) and **Output Adapters**(Upload SBOMs to analysis and security platforms). Currently sbommv support **github** as a input adapter and **interlynk** as aoutput adapter.

## How sbommv Works ?

- Extract SBOMs from Input Systems (GitHub, package registries, etc.)
- Transform or Enrich SBOMs (Future capabilities for format conversion, enrichment)
- Send SBOMs to Output Systems (Security tools, SBOM repositories, compliance platforms)

**Examples**

1. **Generate & Transfer SBOM's for all repositories in Github org of `interlynk-io` to Interlynk SBOM's platform**:
   Generate & Transfer SBOM's from all repositories in the interlynk-io github organization using github apis, and transfer them to interlynk. If interlynk platform does not contain projects it will create them.

   ```bash
   $ export GITHUB_TOKEN=ghp_klgJBxKukyaoWA******
   $ export INTERLYNK_SECURITY_TOKEN=lynk_api******
   $ sbommv transfer --input-adapter=github --in-github-url="https://github.com/interlynk-io" --output-adapter=interlynk --out-interlynk-url="http://localhost:3000/lynkapi"
   ```

2. **DRY RUN: Transfer SBOM from the latest version of sbomqs to interlynk platform**:
  This will look for the latest release of the repository and check if SBOMs are generated, in dry-run mode, it will just iterate the sboms found, and check if login to the output adapter works.

   ```bash
   export GITHUB_TOKEN=ghp_klgJBxKukyaoWA******
   export INTERLYNK_SECURITY_TOKEN=lynk_api******
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

