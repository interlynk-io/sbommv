# sbommv

sbommv - Your primary tool to transfer SBOM's between different systems.

sbommv is designed to allow transfer sboms across systems. The tool supports input, translation, enrichment & output adapters which allow it to be extensible in the future. Input adapters are responsbile to interface with services and provide various methods to extract sboms. The output adapters handles all the complexity related to uploading sboms. 

## Why sbommv? The Motivation Behind Its Creation

### The Problem: Managing SBOMs Across Systems

A Software Bill of Materials (SBOM) plays a crucial role in software supply chain security, compliance, and vulnerability management. However, organizations face a key challenge: **How do you efficiently move SBOMs between different systems?**.

There are two major categories of systems dealing with SBOMs: SBOM Sources (Input Systems) and SBOM Consumers (Output Systems). Manually fetching SBOMs from input systems and uploading them to output systems is: Time-consuming, Error-prone and Difficult to scale.

### The Solution: Automating SBOM Transfers(sbommv)

sbommv is designed to move SBOMs between systems effortlessly. It provides: **Input Adapters**(Fetch SBOMs from different sources) and **Output Adapters**(Upload SBOMs to analysis and security platforms). Currently sbommv support **github** as a input adapter and **interlynk** as aoutput adapter.

## How sbommv Works ?

- Extract SBOMs from Input Systems (GitHub, package registries, etc.)
- Transform or Enrich SBOMs (Future capabilities for format conversion, enrichment)
- Send SBOMs to Output Systems (Security tools, SBOM repositories, compliance platforms)

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

## License 

### Conversion Adapters

## SPDX -> CDX 

## CDX -> SPDX
