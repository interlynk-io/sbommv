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
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"github.com/interlynk-io/sbommv/pkg/logger"
	"github.com/interlynk-io/sbommv/pkg/tcontext"
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
	ApiURL    string
	token     string
	client    *http.Client
	ProjectID string
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
		ApiURL:    config.APIURL,
		token:     config.Token,
		ProjectID: config.ProjectID,
		client: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// SetProjectID updates the project ID for the client
func (c *Client) SetProjectID(projectID string) {
	c.ProjectID = projectID
}

// UploadSBOM uploads a single SBOM from memory to Interlynk
func (c *Client) UploadSBOM(ctx *tcontext.TransferMetadata, sbomData []byte) error {
	if len(sbomData) == 0 {
		return fmt.Errorf("SBOM data is empty")
	}

	// Create a context-aware request with appropriate timeout
	req, err := c.createUploadRequest(ctx, sbomData)
	if err != nil {
		return fmt.Errorf("preparing request: %w", err)
	}

	// Execute request with retry logic
	return c.executeUploadRequest(ctx, req)
}

func (c *Client) createUploadRequest(ctx *tcontext.TransferMetadata, sbomData []byte) (*http.Request, error) {
	const uploadMutation = `
        mutation uploadSbom($doc: Upload!, $projectId: ID!) {
            sbomUpload(input: { doc: $doc, projectId: $projectId }) {
                errors
            }
        }
    `

	// Prepare multipart form data
	body, writer, err := c.prepareMultipartForm(sbomData, uploadMutation)
	if err != nil {
		return nil, err
	}

	// Create request with context
	req, err := http.NewRequestWithContext(ctx.Context, "POST", c.ApiURL, body)
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

func (c *Client) prepareMultipartForm(sbomData []byte, query string) (*bytes.Buffer, *multipart.Writer, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	// Add GraphQL operations
	operations := map[string]interface{}{
		"query": strings.TrimSpace(strings.ReplaceAll(query, "\n", " ")),
		"variables": map[string]interface{}{
			"projectId": c.ProjectID,
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

	// Add SBOM data as a file in-memory
	part, err := writer.CreateFormFile("0", "sbom.json")
	if err != nil {
		return nil, nil, fmt.Errorf("creating form file: %w", err)
	}
	if _, err := part.Write(sbomData); err != nil {
		return nil, nil, fmt.Errorf("writing SBOM content: %w", err)
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

func (c *Client) executeUploadRequest(ctx *tcontext.TransferMetadata, req *http.Request) error {
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
func (c *Client) CreateProjectGroup(ctx *tcontext.TransferMetadata, name, description string, enabled bool) (string, error) {
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

	if c.ApiURL == "" {
		c.ApiURL = "http://localhost:3000/lynkapi"
	}
	req, err := http.NewRequestWithContext(ctx.Context, "POST", c.ApiURL, bytes.NewReader(body))
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

	logger.LogDebug(ctx.Context, "Raw message body", "response", string(respBody))

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
