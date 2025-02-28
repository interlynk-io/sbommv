# Conversion Layer ?

## Overview

Imagine moving a Software Bill of Materials (SBOM) from one system to another —smooth, right? That’s sbommv’s mission, a tool we’ve crafted to transfer SBOMs seamlessly between platforms. But what happens when those systems speak different SBOM languages, like SPDX and CycloneDX? In this blog, we’ll unpack why sbommv needs a conversion layer to handle these mismatches, how we tapped protobom—a Go library promising a universal SBOM representation—to power it, and how protobom’s goals match our needs. Then we’ll dig into the real-world challenges we’ve faced with protobom@v0.5.1, like missing fields and messy mappings, that trip up conversions for tools like Dependency-Track. Buckle up—we’re unpacking the bumps on the road to smooth SBOM transfers!

## What’s a Conversion Layer?

As the name suggests, "conversion" means transforming something from one form to another. In the world of Software Bill of Materials (SBOMs), a conversion layer’s job is to translate an SBOM from one format—like SPDX—to another, such as CycloneDX. But why does this matter? Let’s dive into this through the lens of sbommv, a tool we’ve been working on to streamline SBOM workflows.

## Why Does sbommv Need a Conversion Layer?

Simply put, sbommv is a tool designed to move SBOMs seamlessly from one system to another. We call the "source" or "input" systems from where it pulls SBOMs and the ones to which it pushes SBOMs is known as "output" or "target," or "destination" systems. 

### The catch?

> These systems often don’t speak the same SBOM language. With two dominant standards—SPDX and CycloneDX—there’s a good chance the input system’s format won’t match the output system’s expectations.

Take a real-world example: GitHub (an input system) provides SBOMs in SPDX format via its Dependency Graph API or release assets, while Dependency-Track (an output system) only accepts CycloneDX. This mismatch is where sbommv steps in. Its core mission is to make SBOM movement smooth and seamless, so it has to bridge that format gap. Without a conversion layer, SBOMs would get stuck at the doorstep of incompatible systems—defeating the whole point of a transfer tool.

## Why Choose Protobom for Conversion?

Enter protobom, a Go-based library we picked for sbommv’s conversion layer. 

### Why?

> It promises a universal SBOM representation using Protocol Buffers—a format-neutral way to store SBOM data, no matter if it’s SPDX, CycloneDX, or something else.

Here’s how it works:

- **Universal Bucket**: Protobom defines a common structure (sbom.Document) with fields like Metadata (e.g., Id, Name, Version), NodeList.Nodes (for packages and files), and NodeList.Edges (for relationships). Think of it as a big, flexible bucket that can hold any SBOM’s contents.

- **Read and Write**: It reads SBOMs from SPDX or CycloneDX using unserializers (e.g., spdx23.go, cyclonedx.go), stores them in this bucket, then writes them out to the target format using serializers (e.g., serializer_cdx.go).

- **Goals Match**: Protobom’s objectives—zero data loss, support for multiple formats, and extensibility—line up with sbommv’s need to move SBOMs across systems without breaking a sweat.

We chose protobom because it’s lightweight (minimal dependencies), Go-native (fits sbommv’s stack), and aims to solve the exact problem of format interoperability we face. But as we’ve dug in, we’ve hit some bumps that show protobom@v0.5.1 isn’t quite there yet.

## How Protobom’s Goals Align with sbommv’s Conversion Needs

Protobom’s goals sound like a perfect fit for sbommv’s conversion layer:

- Format-Neutral Representation: sbommv needs to handle any input SBOM—Protobom’s sbom.Document promises that.

- Multiple Format Support: We fetch from GitHub (SPDX) and push to Dependency-Track (CycloneDX)—Protobom claims to ingest and export both.

- Zero Data Loss: We don’t want to lose package details or dependencies—Protobom aims to keep everything intact.

In theory, protobom is the dream teammate for sbommv—a universal translator smoothing out the SBOM journey. But in practice, we’ve found some cracks.

## Challenges while converting from one SBOM to another ?

### Messy Mappings Between SPDX and CycloneDX

- Mapping between SPDX and CycloneDX is a mess. Take "licenseConcluded" and "licenseDeclared"—they land in Node.licenses, then get squeezed into CycloneDX’s single "licenses" field, often as invalid ids like "NOASSERTION" (which Dependency-Track rejects). Going the other way, CycloneDX’s "signatures" or "pedigree" don’t fit in SPDX—protobom drops them because there’s no matching field. This mismatch means data gets lost or mangled, whether it’s SPDX’s "relationships" to CycloneDX’s "dependencies" or CycloneDX’s "vulnerabilities" with no SPDX equivalent.

These gaps mean data loss—whether it’s SPDX’s "PackageVerificationCode" disappearing in CycloneDX or CycloneDX’s "signatures" vanishing in SPDX. Protobom’s universal bucket isn’t big enough yet, and its mapping rules don’t bridge the format divide cleanly.
