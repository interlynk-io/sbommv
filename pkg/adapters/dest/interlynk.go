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

package dest

import (
	"context"
	"net/http"
	"time"

	"github.com/interlynk-io/sbommv/pkg/logger"
	"github.com/interlynk-io/sbommv/pkg/mvtypes"
	"github.com/interlynk-io/sbommv/pkg/target/interlynk"
)

// InterlynkAdapter implements OutputAdapter for the Interlynk platform
type InterlynkAdapter struct {
	projectID string
	baseURL   string
	apiKey    string
	client    *http.Client
	options   OutputOptions
}

// NewInterlynkAdapter creates a new Interlynk adapter
func NewInterlynkAdapter(config mvtypes.Config) *InterlynkAdapter {
	url := config.DestinationConfigs["url"].(string)
	projectID := config.DestinationConfigs["projectID"].(string)
	token := config.DestinationConfigs["token"].(string)
	// if config.BaseURL == "" {
	// 	config.BaseURL = "https://api.interlynk.io" // default URL
	// }

	return &InterlynkAdapter{
		projectID: projectID,
		baseURL:   url,
		apiKey:    token,
		client:    &http.Client{},
		// options:   config.OutputOptions,
	}
}

// InterlynkAdapter implements UploadSBOMs. Hence it implement OutputAdapter.
func (a *InterlynkAdapter) UploadSBOMs(ctx context.Context, sboms []string) error {
	// Initialize Interlynk client
	client := interlynk.NewClient(interlynk.Config{
		Token:     a.apiKey,
		ProjectID: a.projectID,
	})

	// Initialize upload service
	uploadService := interlynk.NewUploadService(client, interlynk.UploadOptions{
		MaxAttempts:   3,
		MaxConcurrent: 1,
		RetryDelay:    time.Second,
	})

	// Upload SBOMs
	results := uploadService.UploadSBOMs(ctx, sboms)

	// Log results
	for _, result := range results {
		if result.Error != nil {
			logger.LogInfo(ctx, "Failed to upload SBOMs", "response", result.Error)
		} else {
			logger.LogInfo(ctx, "SBOM uploaded successfully", "file", result.Path)
		}
	}
	return nil
}
