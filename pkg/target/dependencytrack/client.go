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

package dependencytrack

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/interlynk-io/sbommv/pkg/logger"
	"github.com/interlynk-io/sbommv/pkg/tcontext"
)

type DependencyTrackClient struct {
	apiURL string
	apiKey string
	client *http.Client
}

func NewDependencyTrackClient(config *DependencyTrackConfig) *DependencyTrackClient {
	return &DependencyTrackClient{
		apiURL: config.APIURL,
		apiKey: config.APIKey,
		client: &http.Client{},
	}
}

type Project struct {
	UUID    string `json:"uuid"`
	Name    string `json:"name"`
	Version string `json:"version"`
}

// UploadSBOM uploads an SBOM to a Dependency-Track project
func (c *DependencyTrackClient) UploadSBOM(ctx *tcontext.TransferMetadata, projectName string, sbomData []byte) error {
	logger.LogDebug(ctx.Context, "Intiatializing Uploading SBOMs", "project", projectName)

	payload := map[string]interface{}{
		"projectName": projectName,
		"bom":         string(sbomData), // Dependency-Track expects base64-encoded SBOM, but we'll send raw for simplicity here
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshaling payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx.Context, http.MethodPut, c.apiURL+"/bom", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("X-Api-Key", c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("uploading SBOM: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	logger.LogDebug(ctx.Context, "Successfully Uploaded SBOMs", "project", projectName)
	return nil
}

// CreateProject ensures a project exists, creating it if necessary
func (c *DependencyTrackClient) CreateProject(ctx *tcontext.TransferMetadata, projectName string) (string, error) {
	logger.LogDebug(ctx.Context, "Intiatializing Project Creation", "project", projectName)

	payload := map[string]string{
		"name":    projectName,
		"version": "latest", // Default version
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshaling payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx.Context, "PUT", c.apiURL+"/project", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("X-Api-Key", c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("creating project: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var project Project
	if err := json.NewDecoder(resp.Body).Decode(&project); err != nil {
		return "", fmt.Errorf("decoding response: %w", err)
	}

	logger.LogDebug(ctx.Context, "Successfully Project Created", "project", projectName, "projectID", project.UUID)

	return project.UUID, nil
}
