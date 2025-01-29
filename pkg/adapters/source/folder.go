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

	"github.com/interlynk-io/sbommv/pkg/mvtypes"
)

// FolderAdapter implements InputAdapter specifically for directory sources
// with configurable concurrency and recursive options
type FolderAdapter struct {
	root      string
	options   InputOptions
	recursive bool
}

// NewFolderAdapter creates a new folder-based adapter with concurrent processing
func NewFolderAdapter(config mvtypes.Config) *FolderAdapter {
	path := config.SourceConfigs["url"].(string)

	// // Set reasonable default for concurrent operations
	// if config.InputOptions.MaxConcurrent <= 0 {
	// 	config.InputOptions.MaxConcurrent = 5
	// }

	return &FolderAdapter{
		root: path,
		// options:   config.InputOptions,
		recursive: false,
	}
}

// GetSBOMs implements InputAdapter for FolderAdapter
func (a *FolderAdapter) GetSBOMs(ctx context.Context) (map[string][]string, error) {
	// TODO: Implement Interlynk API integration
	return nil, fmt.Errorf("not implemented")
}
