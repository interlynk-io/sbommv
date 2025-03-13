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

package iterator

import (
	"context"
	"io"

	"github.com/interlynk-io/sbommv/pkg/converter"
	"github.com/interlynk-io/sbommv/pkg/logger"
	"github.com/interlynk-io/sbommv/pkg/tcontext"
)

// SBOM represents a single SBOM file
type SBOM struct {
	Path      string // File path (empty if stored in memory)
	Data      []byte // SBOM data stored in memory (nil if using Path)
	Namespace string // It could be Repo, or Dir (helps track multi-repo or multi-folder processing)
	Version   string // Version of the SBOM (e.g., "latest" or "v1.2.3")
	Branch    string // github repo main, master, or any specific branch
}

// SBOMIterator provides a way to lazily fetch SBOMs one by one
type SBOMIterator interface {
	Next(ctx tcontext.TransferMetadata) (*SBOM, error) // Fetch the next SBOM
}

// MemoryIterator is an iterator that iterates over a preloaded slice of SBOMs.
type MemoryIterator struct {
	sboms []*SBOM
	index int
}

// NewMemoryIterator creates a new MemoryIterator from a slice of SBOMs.
func NewMemoryIterator(sboms []*SBOM) SBOMIterator {
	return &MemoryIterator{
		sboms: sboms,
		index: 0,
	}
}

// Next retrieves the next SBOM in memory.
func (it *MemoryIterator) Next(ctx tcontext.TransferMetadata) (*SBOM, error) {
	if it.index >= len(it.sboms) {
		return nil, io.EOF // No more SBOMs left
	}

	sbom := it.sboms[it.index]
	it.index++
	return sbom, nil
}

type ConvertedIterator struct {
	inner        SBOMIterator
	targetFormat converter.FormatSpec
}

func NewConvertedIterator(inner SBOMIterator, targetFormat converter.FormatSpec) *ConvertedIterator {
	return &ConvertedIterator{
		inner:        inner,
		targetFormat: targetFormat,
	}
}

func (ci *ConvertedIterator) Next(ctx tcontext.TransferMetadata) (*SBOM, error) {
	sbom, err := ci.inner.Next(ctx)
	if err != nil {
		if err == io.EOF {
			return nil, io.EOF // EOF for one-time mode
		}
		if err == context.Canceled || err == context.DeadlineExceeded {
			logger.LogDebug(ctx.Context, "Iterator stopped due to context cancellation")
			return nil, err // Exit for daemon mode
		}
		logger.LogInfo(ctx.Context, "Error retrieving SBOM from inner iterator", "error", err)
		return nil, err
	}
	convertedData, err := converter.ConvertSBOM(ctx, sbom.Data, ci.targetFormat)
	if err != nil {
		logger.LogInfo(ctx.Context, "Failed to convert SBOM", "file", sbom.Path, "error", err)
		return sbom, nil
	}
	sbom.Data = convertedData
	return sbom, nil
}
