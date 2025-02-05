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

package adapter

import (
	"context"
	"fmt"

	"github.com/interlynk-io/sbommv/pkg/iterator"
	"github.com/interlynk-io/sbommv/pkg/logger"
	"github.com/interlynk-io/sbommv/pkg/source/github"
	"github.com/interlynk-io/sbommv/pkg/target/interlynk"
	"github.com/spf13/cobra"
)

// AdapterConfig holds configuration for any adapter (input or output)
type AdapterConfig struct {
	AdapterType string // "github" or "interlynk"

	// github adapter speciic field
	RepoURL     string
	Branch      string
	Version     string
	Owner       string
	Repo        string
	Method      string
	GithubToken string

	// interlynk adapter speciic field
	URL       string
	ProjectID string
	Token     string
}

type AdapterType string

const (
	GithubAdapterType    AdapterType = "github"
	InterlynkAdapterType AdapterType = "interlynk"
)

// Adapter defines the interface for all adapters
type Adapter interface {
	AddCommandParams(cmd *cobra.Command)                                   // Adds CLI flags to the command
	ParseAndValidateParams(cmd *cobra.Command) error                       // Parses & validates input params
	FetchSBOMs(ctx context.Context) (iterator.SBOMIterator, error)         // Fetch SBOMs lazily using iterator
	UploadSBOMs(ctx context.Context, iterator iterator.SBOMIterator) error // Outputs SBOMs (uploading)
}

// NewAdapter initializes and returns the correct adapter
func NewAdapter(ctx context.Context, adapterType string) (Adapter, error) {
	logger.LogInfo(ctx, "Initializing adapter", "adapterType", adapterType)

	switch AdapterType(adapterType) {
	case GithubAdapterType:
		return &github.GitHubAdapter{}, nil
	case InterlynkAdapterType:
		return &interlynk.InterlynkAdapter{}, nil
	default:
		return nil, fmt.Errorf("unsupported adapter type: %s", adapterType)
	}
}
