
# 🚀 Getting Started with SBOMMV

## What is SBOMMV?

`sbommv` is a CLI tool designed to **transfer SBOMs (Software Bill of Materials) between systems**. It fetches SBOMs from **input (source) systems** like **GitHub** and uploads them to **output (destination) systems** like **Interlynk**.  

## **Prerequisites**  

Before getting started, ensure you have:  

- **Go installed** (if you plan to build from source) → [Download Go](https://go.dev/dl/)  
- **Homebrew (Mac/Linux users)** → Recommended for easy installation  
- **Interlynk Account & Security Token** (if uploading SBOMs to Interlynk)  
- **GitHub Token** (if fetching SBOMs using GitHub API)  

## 1. Installation

### Install sbommv via Homebrew (Recommended for macOS & Linux)

```bash
brew tap interlynk-io/sbommv
brew install sbommv
```

### Install sbommv via Binary (Linux/macOS/Windows)

Download the latest release from: [SBOMMV Releases](https://github.com/interlynk-io/sbommv/releases)  

```bash
# Example for Linux (AMD64)
wget https://github.com/interlynk-io/sbommv/releases/latest/download/sbommv-linux-amd64
chmod +x sbommv-linux-amd64
mv sbommv-linux-amd64 /usr/local/bin/sbommv
sbommv help
```

### **Build from Source (For Developers)**  

```bash
git clone https://github.com/interlynk-io/sbommv.git
cd sbommv
make build
./build/sbommv help
```

Now, you have installed sbommv, now let's configure interlynk.

## 2. Configuring Interlynk Authentication

If you are using **Interlynk as the output adapter**, authentication setup is **required** before proceeding. Follow these steps to configure it:

### a. Create an Interlynk Account → [Sign Up](https://app.interlynk.io/auth)  

- Click on **Register**.  
- Sign up using **Google** or **GitHub OAuth**.  
- Once registered and verified, you will be redirected to the **Interlynk platform**.  
- Now, click on **Register Organization**.  

  ![Register Organization](image.png)

- Click on **Save** to finalize the organization setup.  

  ![Save Organization](image-1.png)  

- Your organization has been successfully created!  
- You will now be redirected to the **Interlynk homepage**.  

  ![Interlynk Homepage](image-2.png)  

- On the **right sidebar**, click on **Products**. By default, a product named **"Demo Product"** will be available. All SBOMs will be uploaded to newly created products.  

### b. Generate a Security Token → [Get Token](https://app.interlynk.io/vendor/settings?tab=security%20tokens)  

- Click [here](https://app.interlynk.io/vendor/settings?tab=security%20tokens) to access the security token page.  

  ![Security Token Page](image-3.png)  

- Click on the **"+" button** to generate a new token.  
- Enter the **Token Name** as:  **INTERLYNK_SECURITY_TOKEN**
- Click on **Create** button.  

  ![Create Security Token](image-4.png)  

- Copy the generated token and **export it in your terminal**.  

### c. Export the Token

Run the following command to store the security token as an environment variable:  

```bash
export INTERLYNK_SECURITY_TOKEN="lynk_live_M8kkekkqiGeeuieeLfT1n25WfdmeZ9XhyAy4DUd"
```

## 3. Configuring Github Token

- if fetching SBOMs using GitHub API, then you require a GITHUB_TOKEN. Create one from here and export it

```bash
export GITHUB_TOKEN="ghp_NB23OlUBDb0qWt9emqY5eueii3o0032UL5g" 
```

## 4. Try out sbommv

### a. perform in dry-run to see what's happening

For say, you want to Fetch SBOMs from the latest release of a Github repository and then upload it to Interlynk.

```bash
sbommv transfer --input-adapter=github --in-github-url="https://github.com/sigstore/cosign" \
                --in-github-method=release --output-adapter=interlynk --out-interlynk-url="https://api.interlynk.io/lynkapi" --dry-run
```

### b. Now execute withiut `dry-run` to see what's happening

```bash
sbommv transfer --input-adapter=github --in-github-url="https://github.com/sigstore/cosign" \
                --in-github-method=release --output-adapter=interlynk --out-interlynk-url="https://api.interlynk.io/lynkapi"
```

Now, you  all set to use Interlynk with `sbommv`! 🚀

## **🔹 Next Steps**  

- Now, follow these [examples](https://github.com/interlynk-io/sbommv/blob/main/docs/examples.md#1-basic-transfersingle-repository-github---interlynk).
- Check out more details in the [SBOMMV Documentation](https://github.com/interlynk-io/sbommv)  
- Found an issue? **[Report a Bug](https://github.com/interlynk-io/sbommv/issues)**  

🚀 **Start transferring your SBOMs with SBOMMV today!** 🚀
