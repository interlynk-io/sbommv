
# 🔹 S3 --> Folder Examples 🔹

This guide shows how to use `sbommv` to transfer SBOMs from a cloud storage **S3** to a **folder** using input/output adapters.

## 📘 Overview

`sbommv` is designed to automate SBOM transfers between systems using a modular adapter-based architecture.

In this example:

- **Input (Source) System** → `s3` (pulls SBOMs from s3 storage)
- **Output (Destination) System** → `folder` (uploads SBOMs to local Folders)

### 🗂️ Fetch SBOMs from S3 Bucket

The S3 adapter supports two processing modes:

- **Sequential**: Downloads SBOMs one-by-one (default)
- **Parallel**: Downloads up to 3 SBOMs concurrently for faster processing.

NOTE

Use the `--in-s3-processing-mode` flag to specify `sequential` or `parallel`. The adapter scans the specified bucket and prefix, and detect SBOM file by reading its content via spec.

### 🚀 Saves SBOMs to local Folder

Once SBOMs are fetched, user want to save it to local Folder. To use Folder, provide the **path of the folder**.

## ✅ Transfer SBOMs

### 1. Sequential Fetch from S3 Bucket --> Folder

```bash
sbommv transfer --input-adapter=s3 \
    --in-s3-bucket-name="demo-test-sbom" \
    --in-s3-prefix="dropwizard" \
    --in-s3-region="us-east-1" \
    --in-s3-access-key="AKIA..." \
    --in-s3-secret-key="wJalr..." \
    --output-adapter=folder \
    --out-folder-path="temp"
```

**What this does**:

- Fetches SBOMs sequentially from `s3://demo-test-sbom/dropwizard/`.
- Saves each valid SBOM in a `temp` folder.

If AWS credentials is already configured and located at `~/aws/credentials`, it can auto-fetch those configuration by itself. Therefore, no need to provide external flags like `--in-s3-access-key` and `--in-s3-secret-key`. So updated command will be:

```bash
sbommv transfer --input-adapter=s3 \
    --in-s3-bucket-name="demo-test-sbom" \
    --in-s3-prefix="dropwizard" \
    --in-s3-region="us-east-1" \
    --output-adapter=folder \
    --out-folder-path="temp"
```

### 2. Parallel Fetch from S3 Bucket --> Folder

```bash
sbommv transfer --input-adapter=s3 \
    --in-s3-bucket-name="demo-test-sbom" \
    --in-s3-prefix="dropwizard" \
    --in-s3-region="us-east-1" \
    --in-s3-access-key="AKIA..." \
    --in-s3-secret-key="wJalr..." \
    --processing-mode="parallel" \
    --output-adapter=folder \
    --out-folder-path="temp"
```

**What this does**:

- Fetches SBOMs in parallel (up to 3 concurrent downloads) from `s3://demo-test-sbom/dropwizard/`.
- Saves SBOMs into `temp` folder.

If AWS credentials is already configured and located at `~/aws/credentials`, it can auto-fetch those configuration by itself. Therefore, no need to provide external flags like `--in-s3-access-key` and `--in-s3-secret-key`. So updated command will be:

```bash
sbommv transfer --input-adapter=s3 \
    --in-s3-bucket-name="demo-test-sbom" \
    --in-s3-prefix="dropwizard" \
    --in-s3-region="us-east-1" \
    --processing-mode="parallel" \
    --output-adapter=folder \
    --out-folder-path="temp"
```

## 🔍 Dry-Run Mode (Simulation Only)

Use `--dry-run` to preview what would happen without uploading anything.

### 1. Dry Run with Sequential Fetch --> Folder

```bash
sbommv transfer --input-adapter=s3 \
    --in-s3-bucket-name="demo-test-sbom" \
    --in-s3-prefix="dropwizard" \
    --in-s3-region="us-east-1" \
    --in-s3-access-key="AKIA..." \
    --in-s3-secret-key="wJalr..." \
    --output-adapter=folder \
    --out-folder-path="temp"  \
    --dry-run
```

### 2. Dry Run with Parallel Fetch --> Folder

```bash
sbommv transfer --input-adapter=s3 \
    --in-s3-bucket-name="demo-test-sbom" \
    --in-s3-prefix="dropwizard" \
    --in-s3-region="us-east-1" \
    --in-s3-access-key="AKIA..." \
    --in-s3-secret-key="wJalr..." \
    --in-s3-processing-mode="parallel" \
    --output-adapter=folder \
    --out-folder-path="temp" \
    --dry-run
```

**What this does**:

- Scans SBOMs in `s3://demo-test-sbom/dropwizard/`.
- Lists all SBOMs that would be uploaded
- Displays what would be uploaded (preview mode)

## 🛠️ Configuration Notes

- **Credentials**: Provide `--in-s3-access-key` and `--in-s3-secret-key` for explicit credentials, or rely on AWS default config (e.g., `~/.aws/credentials`, environment variables, IAM roles).
- **Region**: Defaults to `us-east-1` if not specified.

## Conclusion

These examples demonstrate how sbommv streamlines SBOM pull from an S3 bucket and saves to local folder. With support for sequential and parallel processing, explicit credentials, and dry-run previews, sbommv offers a flexible, scriptable solution for automating SBOM workflows in CI/CD pipelines or local environments.
