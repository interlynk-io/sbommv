sbommv - Your primary tool to transfer SBOM's between different systems. 

## Goal
The goal of this tool is to allow for easy transfer of SBOM's between different systems. 

## Summary
sbommv is designed to allow transfer sboms across systems. The tool can be run as a standalone tool or can be integrated into a CI/CD pipeline or server mode. In order to transfer SBOM's the tool will need to construct SBOM's using api or or external tools and then transfer the SBOM to the target system in a format that the target system can understand.

In server mode the expectation is to have a server running that can periodically check for new SBOM's and transfer them to the target system based on cron style configurations. 


## Usage
Security tokens for all systems would need to be provided via ENV variables.

### Create/Move SBOM from github repo to a specified project in interlynk free tier  
sbommv from-url=<repo-url> to-url=<interlynk-url> interlynk-project-id=<project-id> 
e.g. sbommv from-url=github.com/interlynk-io/sbomqs to-url=https://api.interlynk.io/lynkapi --interlynk-project-id=1234 

### Create/Move sboms from all github repos in the organization to Interlynk, auto create outgoing projects on interlynk.
sbommv from-url=<repo-url> to-url=<interlynk-url> interlynk-project-id=<project-id> 
e.g. sbommv from-url=github.com/interlynk-io to-url=https://api.interlynk.io/lynkapi --auto-create-outgoing-projects 

### Create/Move SBOM from github repo to a specified project in interlynk free tier using cdxgen
sbommv from-url=<repo-url> to-url=<interlynk-url> interlynk-project-id=<project-id> 
e.g. sbommv from-url=github.com/interlynk-io/sbomqs to-url=https://api.interlynk.io/lynkapi --interlynk-project-id=1234  --gen-sbom-using=cdxgen
