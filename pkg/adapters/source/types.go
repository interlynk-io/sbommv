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

import "context"

type SBOM struct {
	Content    []byte
	Format     SBOMFormat
	Source     string
	SourceType InputType
}

type SBOMFormat string

const (
	FormatCycloneDX SBOMFormat = "cyclonedx"
	FormatSPDX      SBOMFormat = "spdx"
	FormatUnknown   SBOMFormat = "unknown"
)

type InputType string

const (
	SourceGithub InputType = "github"
)

// Input Adapter defines the interface that all SBOM input adapters must implement
type InputAdapter interface {
	// GetSBOMs retrieves all SBOMs from the source
	GetSBOMs(ctx context.Context) (map[string][]string, error)
}

// InputOptions contains common configuration options for input adapters
type InputOptions struct {
	// MaxConcurrent specifies the maximum number of concurrent operations
	MaxConcurrent int
	// IncludeFormats specifies which SBOM formats to include (empty means all)
	IncludeFormats []SBOMFormat
	// ExcludeFormats specifies which SBOM formats to exclude
	ExcludeFormats []SBOMFormat
}

// AdapterConfig holds configuration for all adapter types.
// Fields are optional depending on the adapter type being created.
type AdapterConfig struct {
	// Common options
	Path         string
	InputOptions InputOptions
	Recursive    bool // For folder adapter

	// GitHub specific
	URL string
	// Owner  string
	// Repo   string
	// Token  string
	Method GitHubMethod

	// S3 specific
	Bucket string
	Prefix string

	// Interlynk specific
	ProjectID string
	BaseURL   string
	APIKey    string
}
