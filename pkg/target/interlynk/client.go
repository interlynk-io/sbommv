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

package interlynk

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/interlynk-io/sbommv/pkg/logger"
)

const uploadMutation = `
mutation uploadSbom($doc: Upload!, $projectId: ID!) {
  sbomUpload(
    input: {
      doc: $doc,
      projectId: $projectId
    }
  ) {
    errors
  }
}
`

type graphQLRequest struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables,omitempty"`
}

const (
	defaultTimeout = 30 * time.Second
	// defaultAPIURL  = "https://api.interlynk.io/lynkapi"
	defaultAPIURL = "http://localhost:3000/lynkapi"
)

// Client handles interactions with the Interlynk API
type Client struct {
	apiURL    string
	token     string
	client    *http.Client
	projectID string
}

// Config holds the configuration for the Interlynk client
type Config struct {
	APIURL      string
	Token       string
	ProjectID   string
	Timeout     time.Duration
	MaxAttempts int
}

// NewClient creates a new Interlynk API client
func NewClient(config Config) *Client {
	if config.APIURL == "" {
		config.APIURL = defaultAPIURL
	}
	if config.Timeout == 0 {
		config.Timeout = defaultTimeout
	}

	return &Client{
		apiURL:    config.APIURL,
		token:     config.Token,
		projectID: config.ProjectID,
		client: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// SetProjectID updates the project ID for the client
func (c *Client) SetProjectID(projectID string) {
	c.projectID = projectID
}

// UploadSBOM uploads a single SBOM file to Interlynk
func (c *Client) UploadSBOM(ctx context.Context, filePath string) error {
	// create new client
	// initiate upload service
	// and then upload SBOMs

	// Validate file existence and size
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("checking file: %w", err)
	}
	if fileInfo.Size() == 0 {
		return fmt.Errorf("file is empty: %s", filePath)
	}

	// Create a context-aware request with appropriate timeout
	req, err := c.createUploadRequest(ctx, filePath)
	if err != nil {
		return fmt.Errorf("preparing request: %w", err)
	}

	// Execute request with retry logic
	return c.executeUploadRequest(ctx, req)
}

func (c *Client) createUploadRequest(ctx context.Context, filePath string) (*http.Request, error) {
	// GraphQL query for file upload
	const uploadMutation = `
        mutation uploadSbom($doc: Upload!, $projectId: ID!) {
            sbomUpload(input: { doc: $doc, projectId: $projectId }) {
                errors
            }
        }
    `

	// Prepare multipart form data
	body, writer, err := c.prepareMultipartForm(filePath, uploadMutation)
	if err != nil {
		return nil, err
	}

	// Create request with context
	req, err := http.NewRequestWithContext(ctx, "POST", c.apiURL, body)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.token))
	req.Header.Set("User-Agent", "sbommv/1.0")
	req.Header.Set("Accept", "application/json")

	return req, nil
}

func (c *Client) prepareMultipartForm(filePath, query string) (*bytes.Buffer, *multipart.Writer, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	// Add operations
	operations := map[string]interface{}{
		"query": strings.TrimSpace(strings.ReplaceAll(query, "\n", " ")),
		"variables": map[string]interface{}{
			"projectId": c.projectID,
			"doc":       nil,
		},
	}

	if err := writeJSONField(writer, "operations", operations); err != nil {
		return nil, nil, err
	}

	// Add map
	if err := writeJSONField(writer, "map", map[string][]string{
		"0": {"variables.doc"},
	}); err != nil {
		return nil, nil, err
	}

	// Add file
	if err := c.attachFile(writer, filePath); err != nil {
		return nil, nil, err
	}

	if err := writer.Close(); err != nil {
		return nil, nil, fmt.Errorf("closing multipart writer: %w", err)
	}

	return &body, writer, nil
}

func writeJSONField(writer *multipart.Writer, fieldName string, data interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshaling %s: %w", fieldName, err)
	}

	if err := writer.WriteField(fieldName, string(jsonData)); err != nil {
		return fmt.Errorf("writing %s field: %w", fieldName, err)
	}
	return nil
}

func (c *Client) attachFile(writer *multipart.Writer, filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("opening file: %w", err)
	}
	defer file.Close()

	part, err := writer.CreateFormFile("0", filepath.Base(filePath))
	if err != nil {
		return fmt.Errorf("creating form file: %w", err)
	}

	if _, err := io.Copy(part, file); err != nil {
		return fmt.Errorf("copying file content: %w", err)
	}

	return nil
}

func (c *Client) executeUploadRequest(ctx context.Context, req *http.Request) error {
	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}

	// Parse response
	var response struct {
		Data struct {
			SBOMUpload struct {
				Errors []string `json:"errors"`
			} `json:"sbomUpload"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("parsing response: %w", err)
	}

	// Check for GraphQL errors
	if len(response.Errors) > 0 {
		return fmt.Errorf("GraphQL error: %s", response.Errors[0].Message)
	}

	// Check for upload errors
	if len(response.Data.SBOMUpload.Errors) > 0 {
		return fmt.Errorf("upload failed: %s", response.Data.SBOMUpload.Errors[0])
	}

	return nil
}

// CreateProjectGroup creates a new project group and returns the default project's ID
func (c *Client) CreateProjectGroup(ctx context.Context, name, description string, enabled bool) (string, error) {
	const createProjectGroupMutation = `
        mutation CreateProjectGroup($name: String!, $desc: String, $enabled: Boolean) {
            projectGroupCreate(
                input: {name: $name, description: $desc, enabled: $enabled}
            ) {
                projectGroup {
                    id
                    name
                    description
                    enabled
                    projects {
                        id
                        name
                    }
                }
                errors
            }
        }
    `

	request := graphQLRequest{
		Query: createProjectGroupMutation,
		Variables: map[string]interface{}{
			"name":    name,
			"desc":    description,
			"enabled": enabled,
		},
	}

	body, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal GraphQL request: %w", err)
	}

	if c.apiURL == "" {
		c.apiURL = "http://localhost:3000/lynkapi"
	}
	req, err := http.NewRequestWithContext(ctx, "POST", c.apiURL, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.token))

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	logger.LogDebug(ctx, "Raw message body", "response", string(respBody))

	var response struct {
		Data struct {
			ProjectGroupCreate struct {
				ProjectGroup struct {
					Projects []struct {
						ID   string `json:"id"`
						Name string `json:"name"`
					} `json:"projects"`
				} `json:"projectGroup"`
				Errors []string `json:"errors"`
			} `json:"projectGroupCreate"`
		} `json:"data"`
	}

	if err := json.Unmarshal(respBody, &response); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(response.Data.ProjectGroupCreate.Errors) > 0 {
		return "", fmt.Errorf("failed to create project group: %s", response.Data.ProjectGroupCreate.Errors[0])
	}

	// Retrieve the first (default) project's ID
	if len(response.Data.ProjectGroupCreate.ProjectGroup.Projects) == 0 {
		return "", fmt.Errorf("no projects found in the created project group")
	}

	return response.Data.ProjectGroupCreate.ProjectGroup.Projects[0].ID, nil
}
