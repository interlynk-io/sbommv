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
	"fmt"

	"github.com/interlynk-io/sbommv/pkg/adapters/source"
)

// AdapterConfig holds configuration for all adapter types.
// Fields are optional depending on the adapter type being created.
type AdapterConfig struct {
	// Common options
	Path         string
	InputOptions source.InputOptions
	Recursive    bool // For folder adapter

	// GitHub specific
	Owner  string
	Repo   string
	Token  string
	Method source.GitHubMethod

	// S3 specific
	Bucket string
	Prefix string

	// Interlynk specific
	ProjectID string
	BaseURL   string
	APIKey    string
}

// NewAdapter creates an appropriate adapter for the given source type.
// It returns an error if the source type is not supported or if required
// configuration is missing.
func NewAdapter(sourceType string, config AdapterConfig) (source.InputAdapter, error) {
	switch sourceType {
	case string(source.SourceFile):
		return source.NewFileAdapter(config.Path, config.InputOptions)

	case string(source.SourceFolder):
		return source.NewFolderAdapter(config.Path, config.Recursive, config.InputOptions)

	case string(source.SourceGithub):
		if config.Owner == "" || config.Repo == "" {
			return nil, fmt.Errorf("GitHub adapter requires owner and repo")
		}
		return source.NewGitHubAdapter(
			config.Owner,
			config.Repo,
			config.Token,
			config.Method,
			config.InputOptions,
		), nil

	case string(source.SourceS3):
		if config.Bucket == "" {
			return nil, fmt.Errorf("S3 adapter requires bucket name")
		}
		return source.NewS3Adapter(
			config.Bucket,
			config.Prefix,
			config.InputOptions,
		)

	case string(source.SourceInterlynk):
		if config.ProjectID == "" {
			return nil, fmt.Errorf("Interlynk adapter requires project ID")
		}
		return source.NewInterlynkAdapter(
			config.ProjectID,
			config.BaseURL,
			config.APIKey,
			config.InputOptions,
		), nil

	default:
		return nil, fmt.Errorf("unsupported source type: %s", sourceType)
	}
}
