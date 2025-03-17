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
