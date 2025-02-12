
# 🔹 SBOMMV Transfer Examples 🔹

## **Overview**  

`sbommv` is a tool designed to transfer SBOMs (Software Bill of Materials) between systems. It operates with two types of systems:  

- **Input (Source) System** → Fetches SBOMs  
- **Output (Destination) System** → Uploads SBOMs  

Currently, `sbommv` supports **GitHub** as an input system and **Interlynk** as an output system.  

### **Fetching SBOMs from GitHub**  

GitHub offers three methods to retrieve SBOMs:

1. **API Method** – Uses GitHub’s Dependency Graph  
2. **Release Method** – Extracts SBOMs from GitHub releases  
3. **Tool Method** – Clones the repository and generates SBOMs using Syft  

### **Uploading SBOMs to Interlynk**

Once SBOMs are fetched, they are uploaded to Interlynk. To use Interlynk, you need to:

1. [Create an Interlynk account](https://app.interlynk.io/auth).
2. Generate an INTERLYNK_SECURITY_TOKEN from [here](https://app.interlynk.io/vendor/settings?tab=security%20tokens).
3. Export the token** before running `sbommv`
    ```bash
    export INTERLYNK_SECURITY_TOKEN="lynk_test_EV2DxRfCfn4wdM8FVaiGkb6ny3KgSJ7JE5zT"
    ```

Now let's have some hands-on with following examples.

## 1. Basic Transfer(Specific Repository): GitHub  → Interlynk

### Github Release Method: Fetch SBOMs from the latest GitHub repo release and upload to Interlynk

```bash
sbommv transfer --input-adapter=github --in-github-url="https://github.com/sigstore/cosign" \
                --output-adapter=interlynk --out-interlynk-url="https://app.interlynk.io/lynkapi"
```

### GitHub API Method (Dependency Graph): Fetch SBOMs using the GitHub API and upload to Interlynk

```bash
sbommv transfer --input-adapter=github --in-github-url="https://github.com/sigstore/cosign" \
                --in-github-method=api --output-adapter=interlynk --out-interlynk-url="https://app.interlynk.io/lynkapi"
```

**NOTE**:

- This is the default github method for fetching SBOMs
- Best when the repository does not publish SBOMs in releases.

### GitHub Tool Method (SBOM Generation Using Syft): Fetch SBOMs by cloning the repository, then generates SBOM using Syft and finally upload to Interlynk

```bash
sbommv transfer --input-adapter=github --in-github-url="https://github.com/sigstore/cosign" \
                --in-github-method=tool --output-adapter=interlynk --out-interlynk-url="https://app.interlynk.io/lynkapi"
```

**NOTE**:

- Useful when neither API nor Release provides SBOMs.

### Fetch SBOMs for a Specific GitHub Branch (Tool Method Only)

```bash
sbommv transfer --input-adapter=github --in-github-url="https://github.com/sigstore/cosign" \
                --in-github-method=tool --in-github-branch="main" \
                --output-adapter=interlynk --out-interlynk-url="https://app.interlynk.io/lynkapi"
```

**NOTE**:

- Only applicable to the `tool` method because releases and API do not support branches.

## 2. Using Dry-Run Mode (No Upload, Just Simulation)**

```bash
sbommv transfer --input-adapter=github --in-github-url="https://github.com/sigstore/cosign" \
                --output-adapter=interlynk --out-interlynk-url="https://app.interlynk.io/lynkapi" --dry-run
```

**NOTE**:

- Useful for previewing the fetched SBOMs without actually uploading them.
- Useful for previewing the SBOMs to be uploaded, project to be created on Interlynk.

## 2. Advanced Transfer(Organization Repos): GitHub → Interlynk

### Github Release Method: Fetch SBOMs from a GitHub Organization by including Specific Repos and then upload to interlynk

```bash
sbommv transfer --input-adapter=github --in-github-url="https://github.com/sigstore" \
                --in-github-method=release --in-github-include-repos=cosign,rekor \
                --output-adapter=interlynk --out-interlynk-url="https://app.interlynk.io/lynkapi"
```

- Above command fetches SBOMs from a `Sigstore` organization only for repositories `cosign`, `rekor`. And then uploaded to seperate projects. All `cosign`, `rekor` SBOMs will be uploaded to `sigstore/cosign` and `sigstore/reko` respectively.

**NOTE**:

- For each repo new project will be created.
- Only fetch SBOMs for `cosign` and `rekor` from the GitHub org.

### Fetch SBOMs using release method from a GitHub Organization by excluding Specific Repos and then upload to interlynk

```bash
sbommv transfer --input-adapter=github --in-github-url="https://github.com/sigstore" \
                --in-github-method=release --in-github-exclude-repos=docs \
                --output-adapter=interlynk --out-interlynk-url="https://app.interlynk.io/lynkapi"
```

- Above command fetches SBOMs from a `Sigstore` organization for all repositories `cosign`, `rekor`, `fulcio` and many more repositories, except `docs` repo. And then all these SBOMs of repo will be uploaded to seperate projects. All `cosign`, `rekor`, `fulcio`, etc  SBOMs will be uploaded to `sigstore/cosign` , `sigstore/reko` , `sigstore/fulcio` and many more respectively.

### Github API Method: Fetch SBOMs from a GitHub Organization by including Specific Repos and then upload to interlynk

```bash
sbommv transfer --input-adapter=github --in-github-url="https://github.com/sigstore" \
                --in-github-method=api --in-github-include-repos=cosign,rekor \
                --output-adapter=interlynk --out-interlynk-url="https://app.interlynk.io/lynkapi"
```

- Above command fetches SBOMs from a `Sigstore` organization only for repositories `cosign`, `rekor`. And then uploaded to seperate projects. All `cosign`, `rekor` SBOMs will be uploaded to `sigstore/cosign` and `sigstore/reko` respectively.

**NOTE**:

- For each repo new project will be created.
- Only fetch SBOMs for `cosign` and `rekor` from the GitHub org.

### Fetch SBOMs using API method from a GitHub Organization by excluding Specific Repos and then upload to interlynk

```bash
sbommv transfer --input-adapter=github --in-github-url="https://github.com/sigstore" \
                --in-github-method=api --in-github-exclude-repos=docs \
                --output-adapter=interlynk --out-interlynk-url="https://app.interlynk.io/lynkapi"
```

- Above command fetches SBOMs from a `Sigstore` organization for all repositories `cosign`, `rekor`, `fulcio` and many more repositories, except `docs` repo. And then all these SBOMs of repo will be uploaded to seperate projects. All `cosign`, `rekor`, `fulcio`, etc  SBOMs will be uploaded to `sigstore/cosign` , `sigstore/reko` , `sigstore/fulcio` and many more respectively.

### Github Tool Method: Fetch SBOMs from a GitHub Organization by including Specific Repos and then upload to interlynk

```bash
sbommv transfer --input-adapter=github --in-github-url="https://github.com/sigstore" \
                --in-github-method=tool --in-github-include-repos=cosign,rekor \
                --output-adapter=interlynk --out-interlynk-url="https://app.interlynk.io/lynkapi"
```

- Above command fetches SBOMs from a `Sigstore` organization only for repositories `cosign`, `rekor`. And then uploaded to seperate projects. All `cosign`, `rekor` SBOMs will be uploaded to `sigstore/cosign` and `sigstore/reko` respectively.

**NOTE**:

- For each repo new project will be created.
- Only fetch SBOMs for `cosign` and `rekor` from the GitHub org.

### Fetch SBOMs using Tool method from a GitHub Organization by excluding Specific Repos and then upload to interlynk

```bash
sbommv transfer --input-adapter=github --in-github-url="https://github.com/sigstore" \
                --in-github-method=tool --in-github-exclude-repos=docs \
                --output-adapter=interlynk --out-interlynk-url="https://app.interlynk.io/lynkapi"
```

- Above command fetches SBOMs from a `Sigstore` organization for all repositories `cosign`, `rekor`, `fulcio` and many more repositories, except `docs` repo. And then all these SBOMs of repo will be uploaded to seperate projects. All `cosign`, `rekor`, `fulcio`, etc  SBOMs will be uploaded to `sigstore/cosign` , `sigstore/reko` , `sigstore/fulcio` and many more respectively.

## 5. Handle Multiple GitHub Repos in an Organization**

```bash
sbommv transfer --input-adapter=github --in-github-url="https://github.com/sigstore" \
                --in-github-method=release --in-github-include-repos=cosign,rekor,fulcio \
                --output-adapter=interlynk --out-interlynk-url="https://app.interlynk.io/lynkapi"
```

## Combine Multiple Flags for Full Customization

```bash
sbommv transfer --input-adapter=github --in-github-url="https://github.com/sigstore/cosign" \
                --in-github-method=tool --in-github-branch="dev" \
                --output-adapter=interlynk --out-interlynk-url="https://app.interlynk.io/lynkapi" \
                --out-interlynk-project-name="cosign-dev" --out-interlynk-project-env="development" --dry-run
```

**NOTE**:

- This fully customizes the transfer, ensuring everything is configured as needed before uploading.**
