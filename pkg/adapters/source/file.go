// Copyright 2025 Interlynk.io
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
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
	"os"
	"path/filepath"
)

// FileAdapter implements InputAdapter for single file and folder sources
type FileAdapter struct {
	path    string
	isDir   bool
	options InputOptions
}

// NewFileAdapter creates a new file-based adapter
func NewFileAdapter(config AdapterConfig) (*FileAdapter, error) {
	info, err := os.Stat(config.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to stat path: %w", err)
	}

	return &FileAdapter{
		path:    config.Path,
		isDir:   info.IsDir(),
		options: config.InputOptions,
	}, nil
}

// GetSBOMs implements InputAdapter
func (a *FileAdapter) GetSBOMs(ctx context.Context) ([]string, error) {
	// TODO: Implement Interlynk API integration
	return nil, fmt.Errorf("not implemented")
}

func isSBOMFile(name string) bool {
	// TODO: Implement better SBOM file detection
	ext := filepath.Ext(name)
	return ext == ".json" || ext == ".xml" || ext == ".spdx" || ext == ".cdx"
}

func detectSBOMFormat(content []byte) SBOMFormat {
	// TODO: Implement format detection based on content
	return FormatUnknown
}
