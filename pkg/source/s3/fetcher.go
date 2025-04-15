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
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/interlynk-io/sbommv/pkg/iterator"
	"github.com/interlynk-io/sbommv/pkg/logger"
	"github.com/interlynk-io/sbommv/pkg/source"
	"github.com/interlynk-io/sbommv/pkg/tcontext"
)

type SBOMFetcher interface {
	Fetch(ctx tcontext.TransferMetadata, config *S3Config) (iterator.SBOMIterator, error)
}

type (
	S3SequentialFetcher struct{}
	S3ParallelFetcher   struct{}
)

func (s *S3ParallelFetcher) Fetch(ctx tcontext.TransferMetadata, config *S3Config) (iterator.SBOMIterator, error) {
	logger.LogDebug(ctx.Context, "Fetching SBOMs Parallelly")

	// implement logic here
	return nil, nil
}

func (s *S3SequentialFetcher) Fetch(ctx tcontext.TransferMetadata, s3cfg *S3Config) (iterator.SBOMIterator, error) {
	logger.LogDebug(ctx.Context, "Fetching SBOMs in ParalSequentiallylel")

	// Load AWS config
	cfg, err := config.LoadDefaultConfig(ctx.Context, config.WithRegion(s3cfg.Region))
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create S3 client
	client := s3.NewFromConfig(cfg)

	// List objects
	resp, err := client.ListObjectsV2(ctx.Context, &s3.ListObjectsV2Input{
		Bucket: aws.String(s3cfg.BucketName),
		Prefix: aws.String(s3cfg.Prefix),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list objects: %w", err)
	}

	// Process objects
	var sbomList []*iterator.SBOM
	for _, obj := range resp.Contents {
		if !source.DetectSBOMsFile(*obj.Key) {
			logger.LogDebug(ctx.Context, "Skipping non-SBOM", "key", *obj.Key)
			continue
		}

		// Download object
		getResp, err := client.GetObject(ctx.Context, &s3.GetObjectInput{
			Bucket: aws.String(s3cfg.BucketName),
			Key:    obj.Key,
		})
		if err != nil {
			logger.LogDebug(ctx.Context, "Failed to download", "key", *obj.Key, "error", err)
			continue
		}

		content, err := io.ReadAll(getResp.Body)
		getResp.Body.Close()
		if err != nil {
			logger.LogDebug(ctx.Context, "Failed to read", "key", *obj.Key, "error", err)
			continue
		}

		// Validate SBOM content
		if !source.IsSBOMFile(content) {
			logger.LogDebug(ctx.Context, "Skipping invalid SBOM", "key", *obj.Key)
			continue
		}

		sbomList = append(sbomList, &iterator.SBOM{
			Path:      *obj.Key,
			Data:      content,
			Namespace: s3cfg.BucketName + "/" + s3cfg.Prefix,
		})
	}

	return NewS3Iterator(sbomList), nil
}
