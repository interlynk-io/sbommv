# `sbommv`: Sbom transfers made easy

[![Go Reference](https://pkg.go.dev/badge/github.com/interlynk-io/sbommv.svg)](https://pkg.go.dev/github.com/interlynk-io/sbommv)
[![Go Report Card](https://goreportcard.com/badge/github.com/interlynk-io/sbommv)](https://goreportcard.com/report/github.com/interlynk-io/sbommv)
![GitHub all releases](https://img.shields.io/github/downloads/interlynk-io/sbommv/total)

`sbommv` is the primary tool for transferring SBOMs between systems —— built to fetch SBOMs from input sources, translate formats, enrich metadata, and push them to output destinations. At its core is a `modular`, `adapter-based` architecture that makes it flexible, scalable, and ready for the future to easily plug in and plug out new tools or systems or platforms.

## SBOM Platform - [Interlynk](https://app.interlynk.io/)

Our SBOM Automation Platform has a new free tier that provides a comprehensive solution to manage SBOMs (Software Bill of Materials) effortlessly. From centralized SBOM storage, built-in SBOM editor, continuous vulnerability mapping and assessment, and support for organizational policies, all while ensuring compliance and enhancing software supply chain security using integrated SBOM quality scores. The free tier is ideal for small teams.
[Try now](https://app.interlynk.io/)

## Getting Started

### Installation

#### Using Prebuilt binaries

```console
https://github.com/interlynk-io/sbommv/releases
```

#### Using Homebrew

```console
brew tap interlynk-io/interlynk
brew install sbommv
```

#### Using Go install

```console
go install github.com/interlynk-io/sbommv@latest
```

#### Developer Installation

This approach involves cloning the repo and building it.

1. Clone the repo `git clone git@github.com:interlynk-io/sbommv.git`
2. `cd` into `sbommv` folder
3. make; make build
4. To test if the build was successful run the following command `./build/sbommv version`

## Quick Start

- Fetch/Pull SBOM from Github and save it to a local folder

```bash
$ sbommv transfer --input-adapter=github \
--in-github-url="https://github.com/interlynk-io/sbomqs" \
--in-github-method="release"  --output-adapter=folder \
--out-folder-path="demo"
```

- Fetch/Pull SBOM from Github and push it to a Dependency-Track

```bash
$ sbommv transfer  --input-adapter=github  \
--in-github-url="https://github.com/interlynk-io/sbommv"  \
--output-adapter=dtrack  \
--out-dtrack-url="http://localhost:8081"
```

**NOTE**: Make sure dependency-track is running locally, if not, [refer](https://github.com/interlynk-io/sbommv/blob/main/examples/setup_dependency_track.md) for setup.

If you have found it interesting soo far, you can show your support via starring ⭐ it.

## What's next 🚀 ??

- [Get started](https://github.com/interlynk-io/sbommv/blob/main/docs/getting_started.md) with sbommv.

## sbommv features

- It allows to fetch SBOMs from github API, Github Release Pages, and folder, refer [here](https://github.com/interlynk-io/sbommv/blob/main/docs/input_adpaters.md) for more..
- It allows to send SBOMs to Dependency-Track, Interlynk, Folde, refer [here](https://github.com/interlynk-io/sbommv/blob/main/docs/output_adapters.md) for more.
- It allows continous folder monitoring and transferring SBOMs continously by running into daemon mode, [refer](https://github.com/interlynk-io/sbommv/blob/main/examples/folder_real_time_monitoring_to_dtrack.md) here for more.
- Internally it uses Protobom library forinter-format conver, read more about it [here](https://github.com/interlynk-io/sbommv/blob/main/docs/conversion_layer.md).

## Data Flow

```text
+---------------------+     +------------------------------+     +----------------------+
|    Input Adapter    | --> |    Enrichment/Translation    | --> |   Output Adapter     |
|-------------------- |     |------------------------------|     |----------------------|
|  - GitHub           |     |  - SBOM Translation*         |     |  - Interlynk         |
|  - BitBucket*       |     |  - Enrichment*               |     |  - Dependency-Track  |
|  - Dependency-Track*|     +------------------------------+     |  - Folder            |
|  - Folder           |                                          |  - GUAC*             |
|  - S3*              |                                          |  - S3*               |
+---------------------+                                          +----------------------+

* Coming Soon
```

If you are looking to integrate more such systems, raise an [issue](https://github.com/interlynk-io/sbommv/issues/new), would love to add them.

## Contributions

We look forward to your contributions, below are a few guidelines on how to submit them

- Fork the repo
- Create your feature/bug branch (`git checkout -b feature/bug`)
- Commit your changes (`git commit -aSm "awesome new feature"`) - commits must be signed
- Push your changes (`git push origin feature/new-feature`)
- Create a new pull-request

## Other Open Source Software tools for SBOMs

- [SBOM Quality Score](https://github.com/interlynk-io/sbomqs) - Quality & Compliance tool
- [SBOM Assembler](https://github.com/interlynk-io/sbomasm) - A tool to compose a single SBOM by combining other SBOMs or parts of them
- [SBOM Search Tool](https://github.com/interlynk-io/sbomagr) - A tool to grep style semantic search in SBOMs
- [SBOM Explorer](https://github.com/interlynk-io/sbomex) - A tool for discovering and downloading SBOMs from a public repository

## Contact

We appreciate all feedback. The best ways to get in touch with us:

- ❓& 🅰️ [Slack](https://join.slack.com/t/sbomqa/shared_invite/zt-2jzq1ttgy-4IGzOYBEtHwJdMyYj~BACA)
- :phone: [Live Chat](https://www.interlynk.io/#hs-chat-open)
- 📫 [Email Us](mailto:hello@interlynk.io)
- 🐛 [Report a bug or enhancement](https://github.com/interlynk-io/sbomex/issues)
- :x: [Follow us on X](https://twitter.com/InterlynkIo)

## Stargazers

If you like this project, please support us by starring ⭐ it.

[![Stargazers](https://starchart.cc/interlynk-io/sbommv.svg)](https://starchart.cc/interlynk-io/sbommv)
