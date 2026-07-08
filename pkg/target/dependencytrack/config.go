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
	"encoding/json"
	"fmt"
)

type DependencyTrackConfig struct {
	APIURL         string
	APIKey         string
	ProjectName    string
	ProjectVersion string // Added field for project version
	Overwrite      bool
}

func NewDependencyTrackConfig(apiURL, version string, overwite bool) *DependencyTrackConfig {
	return &DependencyTrackConfig{
		APIURL:         apiURL,
		ProjectVersion: version,
		Overwrite:      overwite,
	}
}

// String returns a sanitized string representation of the config (for logging)
// The API key is masked for security
func (c *DependencyTrackConfig) String() string {
	apiKeyMasked := maskAPIKey(c.APIKey)
	return fmt.Sprintf("{APIURL:%s APIKey:%s ProjectName:%s ProjectVersion:%s Overwrite:%t}",
		c.APIURL, apiKeyMasked, c.ProjectName, c.ProjectVersion, c.Overwrite)
}

// MarshalJSON returns a JSON representation with masked API key
func (c *DependencyTrackConfig) MarshalJSON() ([]byte, error) {
	type alias DependencyTrackConfig // create alias to avoid infinite recursion
	return json.Marshal(&struct {
		*alias
		APIKey string `json:"APIKey"`
	}{
		alias:  (*alias)(c),
		APIKey: maskAPIKey(c.APIKey),
	})
}

// maskAPIKey masks the API key for logging, showing only first 8 and last 4 characters
func maskAPIKey(apiKey string) string {
	if apiKey == "" {
		return ""
	}
	if len(apiKey) > 12 {
		return apiKey[:8] + "***" + apiKey[len(apiKey)-4:]
	}
	return "***"
}
