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

package source

import (
	"context"
	"fmt"
	"net/http"

	"github.com/interlynk-io/sbommv/pkg/mvtypes"
)

// InterlynkAdapter implements InputAdapter for the Interlynk platform
type InterlynkAdapter struct {
	projectID string
	baseURL   string
	apiKey    string
	client    *http.Client
	options   InputOptions
}

// NewInterlynkAdapter creates a new Interlynk adapter
func NewInterlynkAdapter(config mvtypes.Config) *InterlynkAdapter {
	url := config.SourceConfigs["url"].(string)
	projectID := config.SourceConfigs["projectID"].(string)
	token := config.SourceConfigs["token"].(string)

	// if config.BaseURL == "" {
	// 	config.BaseURL = "https://api.interlynk.io" // default URL
	// }

	return &InterlynkAdapter{
		projectID: projectID,
		baseURL:   url,
		apiKey:    token,
		client:    &http.Client{},
		// options:   config.InputOptions,
	}
}

// GetSBOMs implements InputAdapter
func (a *InterlynkAdapter) GetSBOMs(ctx context.Context) ([]string, error) {
	// TODO: Implement Interlynk API integration
	return nil, fmt.Errorf("not implemented")
}
