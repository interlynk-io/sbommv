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

package s3

import (
	"io"

	"github.com/interlynk-io/sbommv/pkg/iterator"
	"github.com/interlynk-io/sbommv/pkg/tcontext"
)

// S3Iterator implements SBOMIterator
type S3Iterator struct {
	sboms []*iterator.SBOM
	index int
}

// NewS3Iterator creates an S3 iterator
func NewS3Iterator(sboms []*iterator.SBOM) *S3Iterator {
	return &S3Iterator{
		sboms: sboms,
		index: 0,
	}
}

// Next yields the next SBOM
func (it *S3Iterator) Next(ctx tcontext.TransferMetadata) (*iterator.SBOM, error) {
	it.index++
	if it.index >= len(it.sboms) {
		return nil, io.EOF
	}
	return it.sboms[it.index], nil
}
