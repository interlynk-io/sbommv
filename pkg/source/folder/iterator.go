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

package folder

import (
	"context"
	"io"

	"github.com/interlynk-io/sbommv/pkg/iterator"
)

// FolderIterator iterates over SBOMs found in a folder
type FolderIterator struct {
	sboms []*iterator.SBOM
	index int
}

// NewFolderIterator initializes and returns a new FolderIterator
func NewFolderIterator(sboms []*iterator.SBOM) *FolderIterator {
	return &FolderIterator{
		sboms: sboms,
		index: 0,
	}
}

// Next retrieves the next SBOM in the iteration
func (it *FolderIterator) Next(ctx context.Context) (*iterator.SBOM, error) {
	if it.index >= len(it.sboms) {
		return nil, io.EOF
	}

	sbom := it.sboms[it.index]
	it.index++
	return sbom, nil
}
