# Output Adapters and Respective Flags

In sbommv, **output adapters** represent the destination systems where SBOMs are uploaded or stored. These destinations can include:

- SBOM management platforms like **Dependency-Track** and **Interlynk**,  
- Local **folders**,
- Or other **security and analysis tools**.

Output adapters are responsible for **receiving and processing SBOMs** after they've been fetched and optionally transformed.

This document outlines the available output adapters, their CLI flags, and usage examples.

---

## 1. Dependency-Track Adapter

The **Dependency-Track output adapter** uploads SBOMs to a Dependency-Track project. If a project does not exist, it can be automatically created. You must provide a valid `DTRACK_API_KEY` to authenticate with the platform.

### Supported Flags

- `--out-dtrack-url` (required) – URL of the Dependency-Track instance. Defaults to `http://localhost:8081`.  
- `--out-dtrack-project-name` *(Optional)* – Name of the project to upload SBOMs to. If not provided, one is auto-created based on the SBOM’s primary component.
- `--out-dtrack-project-version` *(Optional)* – Version of the project. Defaults to `"latest"` if not specified.

### Authentication

Before running the command, export your Dependency-Track API key:

```bash
export DTRACK_API_KEY="your_api_key_here"
```

Follow this [guide](https://github.com/interlynk-io/sbommv/blob/v0.0.3/examples/github_dtrack_examples.md) to generate Token.

Ensure your team in Dependency-Track has these permissions:

- `BOM_UPLOAD`  
- `PORTFOLIO_MANAGEMENT`  
- `VIEW_PORTFOLIO`  

(Teams with the `Administrators` role have these by default.)

### Usage Examples

```bash
# Upload SBOMs to a project named "xyz" with default version
--out-dtrack-project-name=xyz

# Upload to a specific version
--out-dtrack-project-name=xyz
--out-dtrack-project-version=v0.1.0
```

---

## 2. Interlynk Adapter

The **Interlynk output adapter** uploads SBOMs to the Interlynk Platform. If the specified project does not exist, it will be automatically created. Projects can be assigned to environments such as `"production"` or `"staging"`. By default environment is `"default"`. Authentication is handled via a security token `INTERLYNK_SECURITY_TOKEN`.

### Supported Flags

- `--out-interlynk-url` *(Required)* – URL of the Interlynk API. Defaults to `https://api.interlynk.io/lynkapi`.  
- `--out-interlynk-project-name` *(Optional)* – Name of the target project. If not specified, it will be auto-created.  
- `--out-interlynk-project-env` *(Optional)* – Project environment. Defaults to `"default"`.

### Authentication

Before using this adapter, export your security token:

```bash
export INTERLYNK_SECURITY_TOKEN="your_token_here"
```

Follow this [guide](https://github.com/interlynk-io/sbommv/blob/v0.0.3/docs/getting_started.md#2-configuring-interlynk-authentication) to generate a token.

### Usage Examples

```bash
# Upload to a project named "abc"
--out-interlynk-project-name=abc

# Upload to a project under the "production" environment
--out-interlynk-project-name=abc
--out-interlynk-project-env=production
```

---

## 3. Folder Adapter

The **Folder output adapter** writes SBOMs to a specified directory on the local filesystem. This adapter is useful for debugging, local archiving, or integrating with tools that watch a folder for new SBOMs.

### Supported Flags

- `--out-folder-path` – Path to the folder where SBOMs should be saved.  

### Usage Examples

```bash
# Save SBOMs to folder "temp" in sequential mode
--out-folder-path=temp
--processing-mode="sequential" # global flag

# Save SBOMs to folder "temp" in parallel mode
--out-folder-path=temp
-processing-mode="parallel" # global flag
```

---

## Summary

Output adapters define where your SBOMs go after retrieval. Whether you’re sending them to a cloud platform, a security tool, or simply saving them to disk, sbommv makes it easy to route SBOMs to the right destination through clear, declarative flags.
