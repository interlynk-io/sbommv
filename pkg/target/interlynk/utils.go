// Copyright 2025 Interlynk.io
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package interlynk

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/interlynk-io/sbommv/pkg/logger"
	"github.com/interlynk-io/sbommv/pkg/sbom"
	"github.com/interlynk-io/sbommv/pkg/source"
	"github.com/interlynk-io/sbommv/pkg/tcontext"
)

// ValidateInterlynkConnection chesks whether Interlynk ssytem is up and running
func ValidateInterlynkConnection(url, token string) error {
	ctx := context.Background()

	baseURL, err := genHealthzUrl(url)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL, nil)
	if err != nil {
		return fmt.Errorf("falied to create request for Interlynk: %w", err)
	}

	// INTERLYNK_SECURITY_TOKEN is required here
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to reach Interlynk at %s: %w", baseURL, err)
	}
	defer resp.Body.Close()

	// provided token is invalid
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("invalid API token: authentication failed")
	}

	// interlynk looks to down
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Interlynk API returned unexpected status: %d", resp.StatusCode)
	}

	return nil
}

func genHealthzUrl(rawURL string) (string, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s://%s/healthz", parsedURL.Scheme, parsedURL.Host), nil
}

// formatSetToString converts a map of unique formats to a comma-separated string
func formatSetToString(formatSet map[string]struct{}) string {
	var formats []string
	for format := range formatSet {
		formats = append(formats, format)
	}
	return strings.Join(formats, ", ")
}

// ConstructInterlynkProjectName return project name, in a way that if externally project name is provided, then return it.
// otherwise depends on source. If sboms fetched from github source, then return project name as <organization>/<repo>
// for remaining sources like folder, s3, etc. Return their primary component name + it's version, so unique project created for each SBOM file.
func ConstructInterlynkProjectName(ctx tcontext.TransferMetadata, extProjectName, ownerAndGithubRepoName, sbomPath string, SbomData []byte, source string) string {
	logger.LogDebug(ctx.Context, "Constructing Project Name", "providedProjectName", extProjectName, "ownerAndGithubRepoName", ownerAndGithubRepoName, "source", source, "assetpath", sbomPath)

	if extProjectName != "" {
		return extProjectName
	}

	if source == "github" {
		logger.LogDebug(ctx.Context, "Source is a github")

		if strings.Contains(ownerAndGithubRepoName, "-") {
			return strings.Replace(ownerAndGithubRepoName, "-", "/", 1)
		}
		return ownerAndGithubRepoName
	}

	// if source other than github, then naming would be different
	return GetProjectNameAndVersion(ctx, SbomData, sbomPath)
}

// construct project name from it's primary comp name and it's version by reading sbom content
func GetProjectNameAndVersion(ctx tcontext.TransferMetadata, content []byte, assetPath string) string {
	// ONLY APPLICABLE FOR JSON FILE FORMAT SBOM
	if !source.IsSBOMJSONFormat(content) {
		logger.LogInfo(ctx.Context, "SBOM File Format is not in JSON format")
		return ""
	}

	logger.LogDebug(ctx.Context, "SBOM File Format is in JSON format")
	primaryComp := sbom.ExtractPrimaryComponentName(content)
	logger.LogDebug(ctx.Context, "Project Name and Version", "name", primaryComp.Name, "version", primaryComp.Version)

	if primaryComp.Name != "" && primaryComp.Version != "" {
		return primaryComp.Name + "-" + primaryComp.Version
	} else if primaryComp.Version == "" {
		return primaryComp.Name
	} else {
		return assetPath
	}
}
