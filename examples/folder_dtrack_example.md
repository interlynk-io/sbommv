
# 🔹 Folder --> DependencyTrack Examples 🔹

This guide shows how to use `sbommv` to transfer SBOMs from a local **folder** to a **dtrack**(Dependency-Track) instance using input/output adapters.

## 📘 Overview

`sbommv` is designed to automate SBOM transfers between systems using a modular adapter-based architecture.

In this example:

- **Input (Source) System** → `folder` (reads SBOMs from your local filesystem)
- **Output (Destination) System** → `dtrack` (uploads SBOMs to Dependency-Track)

### 🗂️ Scan SBOMs from local Folder

- By default, sbommv scans only the root of the folder. To include all nested directories, use:
flag, `--in-folder-recursive=true`

### 🚀 Uploading SBOMs to Dependency-Track(dtrack)

Once SBOMs are fetched, they need to be uploaded to DependencyTrack. To setup DependencyTrack, follow this [guide](https://github.com/interlynk-io/sbommv/blob/v0.0.3/examples/setup_dependency_track.md).

## ✅ Transfer SBOMs

### 1. From Root Folder Only (no recursion) --> dtrack

```bash
sbommv transfer \
--input-adapter=folder \
--in-folder-path="temp" \
--output-adapter=dtrack \
--out-dtrack-url="http://localhost:8081"
```

**What this does**:

- Scans SBOMs in the `temp` folder
- Auto-creates a project in Dependency-Track with names like <primary_comp_name:version>
- Uploads each valid SBOM

### 2 From root Folder + sub-directories(recursion) --> dtrack

```bash
sbommv transfer \
--input-adapter=folder \
--in-folder-path="temp" \
--in-folder-recursive=true \
--output-adapter=dtrack \
--out-dtrack-url="http://localhost:8081"
```

**What this does**:

- Scans temp and all subfolders for SBOMs
- Uploads them to Dependency-Track under auto-generated projects

## 🔍 Dry-Run Mode (Simulation Only)

Use --dry-run to preview what would happen without uploading anything.

### 1. Dry Run from Root Folder --> dtrack

```bash
sbommv transfer \
--input-adapter=folder \
--in-folder-path="temp" \
--output-adapter=dtrack \
--out-dtrack-url="http://localhost:8081" \
--dry-run
```

### 2 Dry Run from Folder + Subdirectories --> dtrack

```bash
sbommv transfer \
--input-adapter=folder \
--in-folder-path="temp" \
--in-folder-recursive=true \
--output-adapter=dtrack \
--out-dtrack-url="http://localhost:8081" \
--dry-run
```

**What this does**:

- Scans SBOMs in the `temp` folder
- Lists all SBOMs that would be uploaded
- Displays what would be uploaded (preview mode)

## Conclusion

These examples show how easy it is to move SBOMs from a local folder into Dependency-Track using sbommv. Whether you're dealing with a flat folder or deeply nested SBOM sets, sbommv makes the transfer process seamless, scriptable, and CI/CD-ready.
