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

package s3

import (
	"github.com/interlynk-io/sbommv/pkg/iterator"
	"github.com/interlynk-io/sbommv/pkg/logger"
	"github.com/interlynk-io/sbommv/pkg/tcontext"
)

type SBOMFetcher interface {
	Fetch(ctx tcontext.TransferMetadata, config *S3Config) (iterator.SBOMIterator, error)
}

type (
	S3SequentialFetcher struct{}
	S3ParallelFetcher   struct{}
)

func (s *S3SequentialFetcher) Fetch(ctx tcontext.TransferMetadata, config *S3Config) (iterator.SBOMIterator, error) {
	logger.LogDebug(ctx.Context, "Fetching SBOMs Sequentially")

	// implement logic here
	return nil, nil
}

func (s *S3ParallelFetcher) Fetch(ctx tcontext.TransferMetadata, config *S3Config) (iterator.SBOMIterator, error) {
	logger.LogDebug(ctx.Context, "Fetching SBOMs in Parallel")

	// implement logic here
	return nil, nil
}
