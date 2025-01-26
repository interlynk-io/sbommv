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

import "context"

type AdapterConfig struct {
	OutputOptions OutputOptions

	// Interlynk specific
	ProjectID string
	BaseURL   string
	APIKey    string
}

type OutputType string

const (
	DestInterlynk       OutputType = "interlynk"
	DestDependencyTrack OutputType = "dTrack"
)

// OutputAdapter defines the interface that all SBOM output adapters must implement
type OutputAdapter interface {
	// UploadSBOMs uploads multiple SBOMs to the target system
	UploadSBOMs(ctx context.Context, sboms []string) error
}

// OutputOptions contains common configuration options for output adapters
type OutputOptions struct {
	// MaxConcurrent specifies the maximum number of concurrent upload operations
	MaxConcurrent int

	// RetryAttempts specifies how many times to retry failed uploads
	RetryAttempts int

	// RetryDelay specifies the delay between retry attempts
	RetryDelay int
}
