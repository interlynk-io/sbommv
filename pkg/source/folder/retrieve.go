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
// -------------------------------------------------------------------------

package folder

import (
	"path/filepath"

	"github.com/interlynk-io/sbommv/pkg/logger"
	"github.com/interlynk-io/sbommv/pkg/sbom"
	"github.com/interlynk-io/sbommv/pkg/source"
	"github.com/interlynk-io/sbommv/pkg/tcontext"
)

func getProjectNameAndVersion(ctx tcontext.TransferMetadata, fileName string, content []byte) (string, string) {
	var projectName, projectVersion string

	if source.IsSBOMJSONFormat(content) {

		logger.LogDebug(ctx.Context, "SBOM is in JSON format", "path", fileName)
		primaryComp, err := sbom.ExtractPrimaryComponentName(content)
		if err != nil {
			// when a JSON SBOM has empty primary comp and version, use the file name as project name
			projectName = filepath.Base(fileName)
			projectName = projectName[:len(projectName)-len(filepath.Ext(projectName))]
			projectVersion = "latest"
			logger.LogDebug(ctx.Context, "Failed to parse SBOM for primary component from JSON format SBOM", "path", fileName, "error", err)
		} else {
			projectName, projectVersion = primaryComp.Name, primaryComp.Version
		}
	} else {
		logger.LogDebug(ctx.Context, "SBOM is not in JSON format", "path", fileName)
		projectName = filepath.Base(fileName)
		projectName = projectName[:len(projectName)-len(filepath.Ext(projectName))]

		projectVersion = "latest"
	}

	return projectName, projectVersion
}
