
# 🔹 S3 --> Interlynk Examples 🔹

This guide shows how to use `sbommv` to transfer SBOMs from a cloud storage **S3** to a **interlynk** using input/output adapters.

## 📘 Overview

`sbommv` is designed to automate SBOM transfers between systems using a modular adapter-based architecture.

In this example:

- **Input (Source) System** → `s3` (pulls SBOMs from your s3 storage)
- **Output (Destination) System** → `interlynk` (uploads SBOMs to Interlynk Platform)

### 🗂️ Fetch SBOMs from S3 Bucket

The S3 adapter supports two processing modes:

- **Sequential**: Downloads SBOMs one-by-one (default)
- **Parallel**: Downloads up to 3 SBOMs concurrently for faster processing.

NOTE

Use the `--in-s3-processing-mode` flag to specify `sequential` or `parallel`. The adapter scans the specified bucket and prefix, fetching files with `.json`, `.xml`, or `.sbom` extensions.

### 🚀 Uploading SBOMs to Interlynk Platform

Once SBOMs are fetched, they need to be uploaded to Interlynk. To use Interlynk, you need to:

1. [Create an Interlynk account](https://app.interlynk.io/auth).
2. Generate an **INTERLYNK_SECURITY_TOKEN** from [here](https://app.interlynk.io/vendor/settings?tab=security%20tokens).
3. Export the token before running `sbommv`

    ```bash
    export INTERLYNK_SECURITY_TOKEN="lynk_test_EV2DxRfCfn4wdM8FVaiGkb6ny3KgSJ7JE5zT"
    ```

## ✅ Transfer SBOMs

### 1. Sequential Fetch from S3 Bucket --> Interlynk

```bash
sbommv transfer --input-adapter=s3 \
    --in-s3-bucket-name="demo-test-sbom" \
    --in-s3-prefix="dropwizard" \
    --in-s3-region="us-east-1" \
    --in-s3-access-key="AKIA..." \
    --in-s3-secret-key="wJalr..." \
    --output-adapter=interlynk \
    --out-interlynk-url="http://localhost:3000/lynkapi"
```

**What this does**:

- Fetches SBOMs sequentially from `s3://demo-test-sbom/dropwizard/`.
- Auto-creates a project in Interlynk
- Uploads each valid SBOM

If AWS credentials is already configured and located at `~/aws/credentials`, it can auto-fetch those configuration by itself. Therefore, no need to provide external flags like `--in-s3-access-key` and `--in-s3-secret-key`. So updated command will be:

```bash
sbommv transfer --input-adapter=s3 \
    --in-s3-bucket-name="demo-test-sbom" \
    --in-s3-prefix="dropwizard" \
    --in-s3-region="us-east-1" \
    --output-adapter=interlynk \
    --out-interlynk-url="http://localhost:3000/lynkapi"
```

### 2. Parallel Fetch from S3 Bucket --> Interlynk Platform

```bash
sbommv transfer --input-adapter=s3 \
    --in-s3-bucket-name="demo-test-sbom" \
    --in-s3-prefix="dropwizard" \
    --in-s3-region="us-east-1" \
    --in-s3-access-key="AKIA..." \
    --in-s3-secret-key="wJalr..." \
    --processing-mode="parallel" \
    --output-adapter=interlynk \
    --out-interlynk-url="http://localhost:3000/lynkapi"
```

**What this does**:

- Fetches SBOMs in parallel (up to 3 concurrent downloads) from `s3://demo-test-sbom/dropwizard/`.
- Uploads them to Interlynk under auto-generated projects

If AWS credentials is already configured and located at `~/aws/credentials`, it can auto-fetch those configuration by itself. Therefore, no need to provide external flags like `--in-s3-access-key` and `--in-s3-secret-key`. So updated command will be:

```bash
sbommv transfer --input-adapter=s3 \
    --in-s3-bucket-name="demo-test-sbom" \
    --in-s3-prefix="dropwizard" \
    --in-s3-region="us-east-1" \
    --processing-mode="parallel" \
    --output-adapter=interlynk \
    --out-interlynk-url="http://localhost:3000/lynkapi"
```

## 🔍 Dry-Run Mode (Simulation Only)

Use --dry-run to preview what would happen without uploading anything.

### 1. Dry Run with Sequential Fetch --> Interlynk

```bash
sbommv transfer --input-adapter=s3 \
    --in-s3-bucket-name="demo-test-sbom" \
    --in-s3-prefix="dropwizard" \
    --in-s3-region="us-east-1" \
    --in-s3-access-key="AKIA..." \
    --in-s3-secret-key="wJalr..." \
    --output-adapter=interlynk \
    --out-interlynk-url="http://localhost:3000/lynkapi" \
    --dry-run
```

### 2. Dry Run with Parallel Fetch --> Interlynk Platform

```bash
sbommv transfer --input-adapter=s3 \
    --in-s3-bucket-name="demo-test-sbom" \
    --in-s3-prefix="dropwizard" \
    --in-s3-region="us-east-1" \
    --in-s3-access-key="AKIA..." \
    --in-s3-secret-key="wJalr..." \
    --in-s3-processing-mode="parallel" \
    --output-adapter=interlynk \
    --out-interlynk-url="http://localhost:3000/lynkapi" \
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

These examples demonstrate how sbommv streamlines SBOM transfers from an S3 bucket to Interlynk. With support for sequential and parallel processing, explicit credentials, and dry-run previews, sbommv offers a flexible, scriptable solution for automating SBOM workflows in CI/CD pipelines or local environments.
