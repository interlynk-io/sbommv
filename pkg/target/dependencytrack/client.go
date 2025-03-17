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
	"context"
	"encoding/base64"
	"fmt"
	"time"

	dtrack "github.com/DependencyTrack/client-go"
	"github.com/interlynk-io/sbommv/pkg/logger"
	"github.com/interlynk-io/sbommv/pkg/tcontext"
)

type DependencyTrackClient struct {
	Client *dtrack.Client
}

func NewDependencyTrackClient(config *DependencyTrackConfig) *DependencyTrackClient {
	client, err := dtrack.NewClient(
		config.APIURL,
		dtrack.WithAPIKey(config.APIKey),
		dtrack.WithTimeout(30*time.Second),
	)
	if err != nil {
		logger.LogError(context.Background(), err, "Failed to create Dependency-Track client")
	}

	return &DependencyTrackClient{Client: client}
}

type Project struct {
	UUID    string `json:"uuid"`
	Name    string `json:"name"`
	Version string `json:"version"`
}

func (c *DependencyTrackClient) FindProject(ctx tcontext.TransferMetadata, projectName, projectVersion string) (string, error) {
	logger.LogDebug(ctx.Context, "Finding Project", "project", projectName, "version", projectVersion)

	// dtrack client, retrives all projects
	projects, err := c.Client.Project.GetAll(ctx.Context, dtrack.PageOptions{})
	if err != nil {
		return "", err
	}

	logger.LogDebug(ctx.Context, "Total Number of Projects Available in Dependency Track Platform", "count", projects.TotalCount)

	for _, project := range projects.Items {
		// lookup for the our project name with version
		if project.Name == projectName && project.Version == projectVersion {
			logger.LogDebug(ctx.Context, "Project found", "project", projectName, "version", project.Version, "id", project.UUID)
			return project.UUID.String(), nil
		}
	}

	logger.LogDebug(ctx.Context, "Project not found", "project", projectName, "version", projectVersion)
	return "", nil // Project not found
}

// UploadSBOM uploads an SBOM to a Dependency-Track project
func (c *DependencyTrackClient) UploadSBOM(ctx tcontext.TransferMetadata, projectName, projectVersion string, sbomData []byte) error {
	logger.LogDebug(ctx.Context, "Processing Uploading SBOMs", "project", projectName, "version", projectVersion)

	bomReq := dtrack.BOMUploadRequest{
		ProjectName:    projectName,
		ProjectVersion: projectVersion,
		BOM:            base64.StdEncoding.EncodeToString(sbomData),
	}

	// dtrack client will upload SBOM
	token, err := c.Client.BOM.Upload(ctx.Context, bomReq)
	if err != nil {
		return err
	}

	logger.LogDebug(ctx.Context, "SBOM uploaded successfully", "project", projectName, "token", token)
	return nil
}

// FindOrCreateProject ensures a project exists, returning its UUID after finding or creating project
func (c *DependencyTrackClient) FindOrCreateProject(ctx tcontext.TransferMetadata, projectName, projectVersion string) (string, error) {
	logger.LogDebug(ctx.Context, "Processing finding or Creating Project", "project", projectName, "version", projectVersion)

	// find project using project name and project version
	uuid, err := c.FindProject(ctx, projectName, projectVersion)
	if err != nil {
		return "", fmt.Errorf("finding project: %w", err)
	}
	if uuid != "" {
		logger.LogDebug(ctx.Context, "Project already exists, therefor it wouldn't create a new", "project", projectName, "uuid", uuid)
		return uuid, nil
	}
	logger.LogDebug(ctx.Context, "New project will be created", "name", projectName, "version", projectVersion)

	// create project using project name and project version
	return c.CreateProject(ctx, projectName, projectVersion)
}

// CreateProject creates a new project if it doesn’t exist
func (c *DependencyTrackClient) CreateProject(ctx tcontext.TransferMetadata, projectName, projectVersion string) (string, error) {
	logger.LogDebug(ctx.Context, "Initializing Project Creation", "project", projectName, "version", projectVersion)

	sourceAdapter := ctx.Value("source")

	active := true
	description := "Created & uploaded by sbommv"
	sbommvTag := "sbommv"
	sourceTag := sourceAdapter.(string)

	project := dtrack.Project{
		Name:        projectName,
		Version:     projectVersion,
		Active:      active,
		Description: description,
		Tags: []dtrack.Tag{
			{Name: sbommvTag},
			{Name: sourceTag},
		},
	}
	logger.LogDebug(ctx.Context, "Project is created with following parameters", "name", projectName, "version", projectVersion, "active", active, "description", description, "tag1", sbommvTag, "tag2", sourceTag)

	// dtrack client will create a new project
	created, err := c.Client.Project.Create(ctx.Context, project)
	if err != nil {
		return "", err
	}

	logger.LogDebug(ctx.Context, "New Project created", "project", created.Name, "version", created.Version, "uuid", created.UUID)
	return created.UUID.String(), nil
}
