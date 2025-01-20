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
	"time"
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

const (
	defaultTimeout = 30 * time.Second
	defaultAPIURL  = "https://api.interlynk.io/graphql"
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

// UploadSBOM uploads a single SBOM file to Interlynk
func (c *Client) UploadSBOM(ctx context.Context, filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("opening SBOM file: %w", err)
	}
	defer file.Close()

	// Prepare the GraphQL operation
	operations := map[string]interface{}{
		"query": uploadMutation,
		"variables": map[string]interface{}{
			"doc":       nil,
			"projectId": c.projectID,
		},
	}

	operationsJSON, err := json.Marshal(operations)
	if err != nil {
		return fmt.Errorf("marshaling operations: %w", err)
	}

	// Prepare the map
	mapData := map[string][]string{
		"0": {"variables.doc"},
	}
	mapJSON, err := json.Marshal(mapData)
	if err != nil {
		return fmt.Errorf("marshaling map: %w", err)
	}

	// Create multipart form
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	// Add operations
	if err := writer.WriteField("operations", string(operationsJSON)); err != nil {
		return fmt.Errorf("writing operations field: %w", err)
	}

	// Add map
	if err := writer.WriteField("map", string(mapJSON)); err != nil {
		return fmt.Errorf("writing map field: %w", err)
	}

	// Add file
	part, err := writer.CreateFormFile("0", filepath.Base(filePath))
	if err != nil {
		return fmt.Errorf("creating form file: %w", err)
	}

	if _, err := io.Copy(part, file); err != nil {
		return fmt.Errorf("copying file content: %w", err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("closing multipart writer: %w", err)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, "POST", c.apiURL, &body)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+c.token)

	// Send request
	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("upload failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var result struct {
		Data struct {
			SBOMUpload struct {
				Errors []string `json:"errors"`
			} `json:"sbomUpload"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}

	if len(result.Data.SBOMUpload.Errors) > 0 {
		return fmt.Errorf("upload failed: %v", result.Data.SBOMUpload.Errors)
	}

	return nil
}
