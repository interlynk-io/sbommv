# 🔹 Folder --> S3 Examples 🔹

This guide demonstrates how to use sbommv to transfer SBOMs (Software Bill of Materials) from a local folder to an S3 bucket using input/output adapters.

## 📘 Overview

sbommv is a tool designed to automate SBOM transfers between systems using a modular adapter-based architecture.

In this example:

- **Input (Source) System**→ folder (reads SBOMs from a local folder)
- **Output (Destination) System** → s3 (uploads SBOMs to an S3 bucket)

### 🗂️ Fetch SBOMs from Local Folder

The folder adapter scans a specified local folder for SBOM files. It detects valid SBOMs by reading their content and validating against SBOM specifications (e.g., SPDX, CycloneDX).

### 🚀 Upload SBOMs to S3 Bucket

Once SBOMs are fetched, they are uploaded to an S3 bucket. To use S3 as the output adapter, provide:

- **Bucket name** (--out-s3-bucket-name)
- **Optional prefix** (--out-s3-prefix)
- **Region** (--out-s3-region, defaults to us-east-1 if not specified)
- **Credentials** (via --out-s3-access-key and --out-s3-secret-key, or AWS default config like ~/.aws/credentials)

The S3 adapter supports two processing modes:

- **Sequential**: Uploads SBOMs one-by-one (default).
- **Parallel**: Uploads up to 3 SBOMs concurrently for faster processing.

Use the `--processing-mode` flag to specify `sequential` or `parallel`.

## ✅ Transfer SBOMs

### 1. Basic Transfer: Folder --> S3

```bash
sbommv transfer \
        --input-adapter=folder \
        --in-folder-path="temp-sboms" \
        --output-adapter=s3 \
        --out-s3-bucket-name="demo-test-sbom" \
        --out-s3-prefix="sboms" \
        --out-s3-region="us-east-1" \
        --out-s3-access-key="AKIA..." \
        --out-s3-secret-key="wJalr..." \
        --dry-run
```

**What this does**:

- Scans the `temp-sboms` folder for valid SBOM files.
- Simulates uploading them to `s3://demo-test-sbom/sboms/` with a dry-run preview.

To perform the actual transfer (remove `--dry-run`):

```bash
sbommv transfer \
        --input-adapter=folder \
        --in-folder-path="temp-sboms" \
        --output-adapter=s3 \
        --out-s3-bucket-name="demo-test-sbom" \
        --out-s3-prefix="sboms" \
        --out-s3-region="us-east-1" \
        --out-s3-access-key="AKIA..." \
        --out-s3-secret-key="wJalr..."
```

**NOTE**:

- If AWS credentials are configured in `~/.aws/credentials`, omit `--out-s3-access-key` and `--out-s3-secret-key` or even `--out-s3-region`, if want to stick with default region:

```bash
sbommv transfer \
        --input-adapter=folder\
        --in-folder-path="temp-sboms" \
        --output-adapter=s3\
        --out-s3-bucket-name="demo-test-sbom"\
        --out-s3-prefix="sboms"
```

### 2. Parallel Upload to S3

```bash
sbommv transfer \
        --input-adapter=folder \
        --in-folder-path="temp-sboms" \
        --output-adapter=s3\
        --out-s3-bucket-name="demo-test-sbom"\
        --out-s3-prefix="sboms" \
        --out-s3-region="us-east-1"\
        --out-s3-access-key="AKIA..."\
        --out-s3-secret-key="wJalr..." \
        --processing-mode="parallel"
```

**What this does**:

- Scans the `temp-sboms` folder for valid SBOM files.
- Uploads them to `s3://demo-test-sbom/sboms/` in parallel (up to 3 concurrent uploads).

**NOTE**:

- Use `--processing-mode="parallel"` for faster uploads when handling multiple SBOMs.
- If AWS credentials are configured, omit access/secret keys as shown above.

### 3. Dry-Run Mode (Simulation Only)

```bash
sbommv transfer \
        --input-adapter=folder \
        --in-folder-path="temp-sboms" \
        --output-adapter=s3 \
        --out-s3-bucket-name="demo-test-sbom"\
        --out-s3-prefix="sboms" \
        --out-s3-region="us-east-1" \
        --out-s3-access-key="AKIA..." \
        --out-s3-secret-key="wJalr..." \
        --dry-run
```

**What this does**:

- Scans the `temp-sboms` folder for valid SBOM files.
- Lists SBOMs that would be uploaded to `s3://demo-test-sbom/sboms/` without performing the upload.

## Conclusion

These examples illustrate how sbommv streamlines SBOM transfers from a local folder to an S3 bucket. With support for sequential/parallel uploads, selective subfolder scanning, and dry-run previews, sbommv provides a flexible, scriptable solution for automating SBOM workflows in CI/CD pipelines or local environments.
