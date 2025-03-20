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

func ConstructProjectName(ctx tcontext.TransferMetadata, extProjectName, extProjectVersion, intProjectName, intProjectVersion string, content []byte, source string) (string, string) {
	logger.LogDebug(ctx.Context, "Constructing Project Name and Version", "providedProjectName", extProjectName, "providedProjectVersion", extProjectVersion, "primaryCompName", intProjectName, "primaryCompVersion", intProjectVersion, "source", source)
	if extProjectName != "" {
		return getExplicitProjectVersion(extProjectName, extProjectVersion)
	}
	if source != "folder" {
		logger.LogDebug(ctx.Context, "Source is not folder, instead it's a github")
		return getImplicitProjectVersion(intProjectName, intProjectVersion)
	}
	return getProjectNameAndVersion(ctx, content)
}

// extract primary compo name and it's version by reading sbom content
func getProjectNameAndVersion(ctx tcontext.TransferMetadata, content []byte) (string, string) {
	// ONLY APPLICABLE FOR JSON FILE FORMAT SBOM
	if !source.IsSBOMJSONFormat(content) {
		return "", ""
	}

	logger.LogDebug(ctx.Context, "SBOM File Format is in JSON format")
	primaryComp := sbom.ExtractPrimaryComponentName(content)

	logger.LogDebug(ctx.Context, "Project Name and Version", "name", primaryComp.Name, "version", primaryComp.Version)

	return primaryComp.Name, primaryComp.Version
}

func getExplicitProjectVersion(providedProjectName string, providedProjectVersion string) (string, string) {
	if providedProjectVersion == "" {
		return providedProjectName, "latest"
	}

	return providedProjectName, providedProjectVersion
}

func getImplicitProjectVersion(primaryCompName string, primaryCompVersion string) (string, string) {
	return primaryCompName, primaryCompVersion
}
