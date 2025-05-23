// Copyright 2025 Interlynk.io
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package utils

import (
	"github.com/interlynk-io/sbommv/pkg/logger"
	"github.com/interlynk-io/sbommv/pkg/sbom"
	"github.com/interlynk-io/sbommv/pkg/source"
	"github.com/interlynk-io/sbommv/pkg/tcontext"
)

func ConstructProjectName(ctx tcontext.TransferMetadata, extProjectName, extProjectVersion, ownerRepoGithubName, repoVersion, assetPath string, content []byte, source string) (string, string) {
	logger.LogDebug(ctx.Context, "Constructing Project Name and Version", "providedProjectName", extProjectName, "providedProjectVersion", extProjectVersion, "ownerRepoName", ownerRepoGithubName, "repoVersion", repoVersion, "source", source, "assetpath", assetPath)

	// user provided project name via flag
	if extProjectName != "" {
		return getExplicitProjectVersion(extProjectName, extProjectVersion)
	}

	// no project name provided
	// if source is github, then naming would be different
	if source == "github" {
		logger.LogDebug(ctx.Context, "Source is a github")
		return getImplicitProjectVersion(ownerRepoGithubName, repoVersion, assetPath)
	}

	// if source other than github, then naming would be different
	return getProjectNameAndVersion(ctx, content, assetPath)
}

// construct project name from it's primary comp name and it's version by reading sbom content
func getProjectNameAndVersion(ctx tcontext.TransferMetadata, content []byte, assetPath string) (string, string) {
	// ONLY APPLICABLE FOR JSON FILE FORMAT SBOM
	if !source.IsSBOMJSONFormat(content) {
		logger.LogInfo(ctx.Context, "SBOM File Format is not in JSON format")
		return "", ""
	}

	logger.LogDebug(ctx.Context, "SBOM File Format is in JSON format")
	primaryComp := sbom.ExtractPrimaryComponentName(content)

	if primaryComp.Name == "" {
		if primaryComp.Version == "" {
			return assetPath + "-" + "latest", "latest"
		}
		return assetPath + primaryComp.Version, primaryComp.Version
	}

	if primaryComp.Version == "" {
		return primaryComp.Name + "-" + "latest", "latest"
	}
	logger.LogDebug(ctx.Context, "Project Name and Version", "name", primaryComp.Name, "version", primaryComp.Version)
	return primaryComp.Name + "-" + primaryComp.Version, primaryComp.Version
}

func getExplicitProjectVersion(providedProjectName string, providedProjectVersion string) (string, string) {
	if providedProjectVersion == "" {
		return providedProjectName + "-" + "latest", "latest"
	}

	return providedProjectName + "-" + providedProjectVersion, providedProjectVersion
}

func getImplicitProjectVersion(ownerRepoGithubName, repoVersion, assetName string) (string, string) {
	if assetName == "" {
		return ownerRepoGithubName + "-" + repoVersion, repoVersion
	}
	return ownerRepoGithubName + "-" + repoVersion + "-" + assetName, repoVersion
}
