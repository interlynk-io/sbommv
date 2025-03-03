
# 🔹 Folder --> Interlynk Examples 🔹

Scan SBOM from Folder System(adapter) and upload it to Interlynk System(adapter)

## Overview

`sbommv` is a tool designed to transfer SBOMs (Software Bill of Materials) between systems. It operates with two different systems. In this case:

- **Input (Source) System** → Fetches SBOMs  --> **Folder**
- **Output (Destination) System** → Uploads SBOMs  --> **Interlynk**

### Scan SBOMs from local Folder

- By default SBOM will be scanned from root folder, in order to scans SBOMs from all sub-directories or recursive directories, provide a flag, `--in-folder-recursive=true`

### Uploading SBOMs to Interlynk

Once SBOMs are fetched, they need to be uploaded to Interlynk. To use Interlynk, you need to:

1. [Create an Interlynk account](https://app.interlynk.io/auth).
2. Generate an **INTERLYNK_SECURITY_TOKEN** from [here](https://app.interlynk.io/vendor/settings?tab=security%20tokens).
3. Export the token before running `sbommv`

    ```bash
    export INTERLYNK_SECURITY_TOKEN="lynk_test_EV2DxRfCfn4wdM8FVaiGkb6ny3KgSJ7JE5zT"
    ```

Now let's dive into various use cases and examples.

## 1. Basic Transfer(Single Repository): GitHub  → Interlynk

### 1.1 From root Folder(no recursion)

```bash
sbommv transfer --input-adapter=folder --in-folder-path="temp" \
                 --output-adapter=interlynk --out-interlynk-url="https://api.interlynk.io/lynkapi"
```

- **What this does**:
  - Scan SBOMs from the `temp` directory.
  - And upload the scanned SBOMs to Interlynk

### 1.2 From root Folder as well as sub-directories(recursion)

```bash
sbommv transfer --input-adapter=folder --in-folder-path="temp" --in-folder-recursive=true \
                 --output-adapter=interlynk --out-interlynk-url="https://api.interlynk.io/lynkapi"
```

- **What this does**:
  - Scan SBOMs from the `temp` as well as all it's sub-directories.
  - And upload the scanned SBOMs to Interlynk

## 2. Dry-Run Mode (No Upload, Just Simulation)

### 2.1 From root Folder(no recursion)

```bash
sbommv transfer --input-adapter=folder --in-folder-path="temp" \
                 --output-adapter=interlynk --out-interlynk-url="https://api.interlynk.io/lynkapi" --dry-run
```

- **What this does**:
  - Scan SBOMs in a root dir `temp` (simulates the process).
  - Displays what would be uploaded (preview mode)

### 2.2 From root Folder as well as sub-directories(recursion)

```bash
sbommv transfer --input-adapter=folder --in-folder-path="temp" --in-folder-recursive=true \
                 --output-adapter=interlynk --out-interlynk-url="https://api.interlynk.io/lynkapi" --dry-run
```

- **What this does**:
  - Scan SBOMs from root dir `temp` as well as it's all sub-dir (simulates the process).
  - Displays what would be uploaded (preview mode)

## Conclusion

These examples cover various ways to fetch and upload SBOMs using sbommv. Whether you are fetching SBOMs from a single repo, an entire organization, or using a specific branch, sbommv provides flexibility to handle it efficiently.
