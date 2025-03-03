# 📌 SBOM Transfer Examples using `sbommv`

The `examples/` directory contains **practical examples** of how to use different **input and output adapters** with `sbommv` to transfer **SBOMs (Software Bill of Materials)** between various systems.  

## 🔹 Understanding Input & Output Adapters

`sbommv` operates with two systems:  

1️⃣ **Input (Source) System** → Fetches SBOMs  
2️⃣ **Output (Destination) System** → Uploads SBOMs  

### Supported Adapters in These Examples

| **Input Systems (Fetch SBOMs from...)** | **Output Systems (Upload SBOMs to...)** |
|-----------------------------------------|-----------------------------------------|
| `github` → Fetch from a **GitHub repository** | `dtrack` → Upload to **Dependency-Track** |
| `folder` → Fetch from a **local folder** | `interlynk` → Upload to **Interlynk** |
| | `folder` → Save SBOMs to a **local folder** |

Each example in this directory demonstrates a specific **input-output combination** using `sbommv`.

## 🔹 Example File Naming Convention

Each example follows the format:  

📄 **`<input>_<output>_examples.md`**  

| **File Name** | **Description** |
|--------------|---------------|
| `folder_dtrack_example.md` | Fetch SBOMs from a **local folder** and upload to **Dependency-Track** |
| `folder_interlynk_examples.md` | Fetch SBOMs from a **local folder** and upload to **Interlynk** |
| `github_dtrack_examples.md` | Fetch SBOMs from **GitHub** and upload to **Dependency-Track** |
| `github_folder_examples.md` | Fetch SBOMs from **GitHub** and save to a **local folder** |
| `github_interlynk_examples.md` | Fetch SBOMs from **GitHub** and upload to **Interlynk** |

## 📂 Directory Structure

```bash
examples
├── folder_dtrack_example.md      # Folder → Dependency-Track
├── folder_interlynk_examples.md  # Folder → Interlynk
├── github_dtrack_examples.md     # GitHub → Dependency-Track
├── github_folder_examples.md     # GitHub → Folder
├── github_interlynk_examples.md  # GitHub → Interlynk
├── image-1.png                   # Example images/screenshots
├── image.png                      # Example images/screenshots
└── README.md                      # This file
```

## **🚀 Next Steps**

✅ **Go through the examples**  
✅ **Run `sbommv` commands** in real use cases  
✅ **Contribute additional examples**  

🔹 **Need Help?**

This structured **README** makes it easier for users to **understand, navigate, and use** the examples efficiently. Raise an issue or start a discussion if you'd like any refinements! 😊
