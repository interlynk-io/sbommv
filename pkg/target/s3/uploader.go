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
	"bytes"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/interlynk-io/sbommv/pkg/iterator"
	"github.com/interlynk-io/sbommv/pkg/logger"
	"github.com/interlynk-io/sbommv/pkg/tcontext"
)

type SBOMUploader interface {
	Upload(ctx tcontext.TransferMetadata, config *S3Config, iter iterator.SBOMIterator) error
}

type (
	S3SequentialUploader struct{}
	S3ParallelUploader   struct{}
)

// Upload uploads SBOMs to S3 in parallel
func (u *S3ParallelUploader) Upload(ctx tcontext.TransferMetadata, config *S3Config, iter iterator.SBOMIterator) error {
	logger.LogDebug(ctx.Context, "Writing SBOMs in concurrently", "bucket", config.BucketName, "prefix", config.Prefix)

	totalSBOMs := 0
	successfullyUploaded := 0
	prefix := config.Prefix

	client, err := config.GetAWSClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}

	// add "/" to prefix if not present in the end
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix = prefix + "/"
	}

	// retrieve all SBOMs from iterator
	var sbomList []*iterator.SBOM
	for {
		sbom, err := iter.Next(ctx)
		if err == io.EOF {
			break
		}
		if err != nil {
			logger.LogError(ctx.Context, err, "Error retrieving SBOM from iterator")
			continue
		}
		sbomList = append(sbomList, sbom)
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	const maxConcurrency = 3
	semaphore := make(chan struct{}, maxConcurrency)

	for _, sbom := range sbomList {
		wg.Add(1)
		semaphore <- struct{}{}
		go func(sbom *iterator.SBOM) {
			defer wg.Done()
			defer func() { <-semaphore }()

			key := filepath.Join(prefix, sbom.Path)

			// Upload to S3
			_, err := client.PutObject(ctx.Context, &s3.PutObjectInput{
				Bucket: aws.String(config.BucketName),
				Key:    aws.String(key),
				Body:   bytes.NewReader(sbom.Data),
			})

			mu.Lock()
			totalSBOMs++
			if err != nil {
				logger.LogError(ctx.Context, err, "Failed to upload SBOM", "bucket", config.BucketName, "key", key)
				mu.Unlock()
				return
			}
			successfullyUploaded++
			logger.LogDebug(ctx.Context, "Uploaded SBOM", "bucket", config.BucketName, "key", key, "size", len(sbom.Data))
			mu.Unlock()
		}(sbom)
	}

	wg.Wait()

	logger.LogInfo(ctx.Context, "Upload summary", "total", totalSBOMs, "successful", successfullyUploaded, "failed", totalSBOMs-successfullyUploaded)
	if totalSBOMs == 0 {
		return fmt.Errorf("no SBOMs found to upload")
	}

	return nil
}

func (u *S3SequentialUploader) Upload(ctx tcontext.TransferMetadata, s3cfg *S3Config, iter iterator.SBOMIterator) error {
	logger.LogDebug(ctx.Context, "Writing SBOMs sequentially", "bucketName", s3cfg.BucketName+"prefix", s3cfg.Prefix)
	totalSBOMs := 0
	successfullyUploaded := 0
	bucketPrefix := s3cfg.Prefix

	client, err := s3cfg.GetAWSClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}

	// add "/" to prefix if not present in the end
	if bucketPrefix != "" && !strings.HasSuffix(bucketPrefix, "/") {
		bucketPrefix = bucketPrefix + "/"
	}

	for {
		sbom, err := iter.Next(ctx)
		if err == io.EOF {
			break
		}

		totalSBOMs++
		if err != nil {
			logger.LogError(ctx.Context, err, "Error retrieving SBOM from iterator")
			continue
		}

		key := filepath.Join(bucketPrefix, sbom.Path)

		// Upload to S3
		_, err = client.PutObject(ctx.Context, &s3.PutObjectInput{
			Bucket: aws.String(s3cfg.BucketName),
			Key:    aws.String(key),
			Body:   bytes.NewReader(sbom.Data),
		})
		if err != nil {
			logger.LogError(ctx.Context, err, "Failed to upload SBOM", "bucket", s3cfg.BucketName, "key", key)
			continue
		}

		successfullyUploaded++
		logger.LogDebug(ctx.Context, "Uploaded SBOM", "bucket", s3cfg.BucketName, "key", key, "size", len(sbom.Data))
	}
	return nil
}
