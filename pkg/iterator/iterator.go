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
)

// SBOM represents a single SBOM file
type SBOM struct {
	Path    string // File path (empty if stored in memory)
	Data    []byte // SBOM data stored in memory (nil if using Path)
	Repo    string // Repository URL (helps track multi-repo processing)
	Version string // Version of the SBOM (e.g., "latest" or "v1.2.3")
}

// SBOMIterator provides a way to lazily fetch SBOMs one by one
type SBOMIterator interface {
	Next(ctx context.Context) (*SBOM, error) // Fetch the next SBOM
}
