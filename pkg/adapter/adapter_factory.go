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
	"github.com/interlynk-io/sbommv/pkg/types"
	"github.com/spf13/cobra"
)

// Adapter defines the interface for all adapters
type Adapter interface {
	// Adds CLI flags to the commands
	AddCommandParams(cmd *cobra.Command)

	// Parses & validates input params
	ParseAndValidateParams(cmd *cobra.Command) error

	// Fetch SBOMs lazily using iterator
	FetchSBOMs(ctx context.Context) (iterator.SBOMIterator, error)

	// Outputs SBOMs (uploading)
	UploadSBOMs(ctx context.Context, iterator iterator.SBOMIterator) error
}

// NewAdapter initializes and returns the correct adapter
func NewAdapter(ctx context.Context, adapterType string, role types.AdapterRole) (Adapter, error) {
	logger.LogInfo(ctx, "Initializing adapter", "adapterType", adapterType)

	switch types.AdapterType(adapterType) {

	case types.GithubAdapterType:
		return &github.GitHubAdapter{Role: role}, nil

	case types.InterlynkAdapterType:
		return &interlynk.InterlynkAdapter{Role: role}, nil

	default:
		return nil, fmt.Errorf("unsupported adapter type: %s", adapterType)
	}
}
