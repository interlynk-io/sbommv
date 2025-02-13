# sbommv

sbommv - Your primary tool to transfer SBOM's between different systems.

## Summary

sbommv is designed to allow transfer sboms across systems. The tool supports input, translation, enrichment & output adapters which allow it to be extensible in the future. Input adapters are responsbile to interface with services and provide various methods to extract sboms. The output adapters handles all the complexity related to uploading sboms. 

## 🔹 Why sbommv? The Motivation Behind Its Creation

The **Interlynk platform** is a powerful **SBOM (Software Bill of Materials) compiler**, designed to:  

✅ **Analyze SBOMs** → Extract insights and understand software components  
✅ **Perform Compliance Checks** → Ensure SBOMs meet regulatory and security requirements  
✅ **Manage Vulnerabilities** → Identify risks and track security issues across SBOMs  
✅ **Enforce SBOM-related Policies** → Maintain consistency and governance over SBOMs  

To **leverage these capabilities**, the **Interlynk platform requires SBOMs** to be available for processing. However, the challenge lies in **how SBOMs are supplied to the platform**.  In short remember Interlynk as a SBOM compiler.

### 🚀 The Need for sbommv: Bridging the Gap in SBOM Management

There are **two main ways** to feed SBOMs into Interlynk:  

#### Locally Available SBOMs

For SBOMs **already present on your system**, you can:  

- **Manual Upload** → Directly upload an SBOM through the Interlynk web platform.  
- **pylynk Integration** → Use the `pylynk` CLI tool to automate SBOM uploads from local storage.  

📌 **Limitation:** These methods work **only for SBOMs already generated and stored locally**, requiring **manual effort** or **pre-existing automation**.  

### Externally Available SBOMs (The sbommv Solution)

Many SBOMs **are not locally available** but exist in **external systems** like:  

- **GitHub Repositories** → SBOMs generated in CI/CD pipelines  
- **Public Websites** → Organizations publishing SBOMs externally  
- **Other SBOM Repositories** → Systems storing SBOMs for compliance tracking  

🛑 **The Challenge:**  
Manually fetching SBOMs from **multiple sources** and **uploading them individually** is **slow, repetitive, and error-prone**.

✅ **The sbommv Solution:**  
**sbommv automates the transfer of SBOMs from external sources to Interlynk.** It:  

🔹 **Extracts SBOMs from external systems** (e.g., GitHub)  
🔹 **Handles different GitHub methods** (`release`, `api`, `tool`) for fetching SBOMs  
🔹 **Uploads SBOMs seamlessly to Interlynk**  
🔹 **Supports dry-run mode** to preview SBOMs before actual upload  
🔹 **Ensures interoperability** for better **automation and scalability**  

## What's next 🚀 ??

- **Getting started with sbommv:** <https://github.com/interlynk-io/sbommv/blob/main/docs/getting_started.md>
- **Try out more examples:** <https://github.com/interlynk-io/sbommv/blob/main/docs/examples.md>
- **To get with detailed CLI command and it's flag usage:** <https://github.com/interlynk-io/sbommv/blob/main/docs/flag_usage.md>
- To know more about Input and Output adapters: <https://github.com/interlynk-io/sbommv/blob/main/docs/adapters.md>

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
