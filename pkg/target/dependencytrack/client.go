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
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

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
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

type Project struct {
	UUID    string `json:"uuid"`
	Name    string `json:"name"`
	Version string `json:"version"`
}

func (c *DependencyTrackClient) FindProject(ctx *tcontext.TransferMetadata, projectName, projectVersion string) (string, error) {
	req, err := http.NewRequestWithContext(ctx.Context, "GET", c.apiURL+"/project", nil)
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("X-Api-Key", c.apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetching projects: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var projects []Project
	if err := json.NewDecoder(resp.Body).Decode(&projects); err != nil {
		return "", fmt.Errorf("decoding projects: %w", err)
	}
	for _, p := range projects {
		if p.Name == projectName && p.Version == projectVersion {
			return p.UUID, nil
		}
	}
	return "", nil // Project not found
}

// UploadSBOM uploads an SBOM to a Dependency-Track project
func (c *DependencyTrackClient) UploadSBOM(ctx *tcontext.TransferMetadata, projectName, projectVersion string, sbomData []byte) error {
	payload := map[string]interface{}{
		"projectName":    projectName,
		"projectVersion": projectVersion,
		"bom":            base64.StdEncoding.EncodeToString(sbomData),
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

// FindOrCreateProject ensures a project exists, returning its UUID
func (c *DependencyTrackClient) FindOrCreateProject(ctx *tcontext.TransferMetadata, projectName, projectVersion string) (string, error) {
	uuid, err := c.FindProject(ctx, projectName, projectVersion)
	if err != nil {
		return "", fmt.Errorf("finding project: %w", err)
	}
	if uuid != "" {
		logger.LogDebug(ctx.Context, "Project already exists", "project", projectName, "uuid", uuid)
		return uuid, nil
	}
	return c.CreateProject(ctx, projectName, projectVersion)
}

// CreateProject creates a new project if it doesn’t exist
func (c *DependencyTrackClient) CreateProject(ctx *tcontext.TransferMetadata, projectName, projectVersion string) (string, error) {
	payload := map[string]string{
		"name":    projectName,
		"version": projectVersion,
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
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var project Project
	if err := json.NewDecoder(resp.Body).Decode(&project); err != nil {
		return "", fmt.Errorf("decoding response: %w", err)
	}
	logger.LogDebug(ctx.Context, "Project created or verified", "project", projectName, "uuid", project.UUID)
	return project.UUID, nil
}
