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
	"github.com/interlynk-io/sbommv/pkg/utils"
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

	// space for proper logging
	fmt.Println()

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

			sourceAdapter := ctx.Value("source")
			finalProjectName, _ := utils.ConstructProjectName(ctx, "", "", sbom.Namespace, sbom.Version, sbom.Path, sbom.Data, sourceAdapter.(string))

			key := filepath.Join(prefix, finalProjectName)

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
			logger.LogInfo(ctx.Context, "upload", "success", true, "bucket", config.BucketName, "prefix", config.Prefix, "filename", finalProjectName)

			mu.Unlock()
		}(sbom)
	}

	wg.Wait()

	logger.LogInfo(ctx.Context, "upload", "total", totalSBOMs, "success", successfullyUploaded, "failed", totalSBOMs-successfullyUploaded)
	if totalSBOMs == 0 {
		return fmt.Errorf("no SBOMs found to upload")
	}

	return nil
}

func (u *S3SequentialUploader) Upload(ctx tcontext.TransferMetadata, s3cfg *S3Config, iter iterator.SBOMIterator) error {
	logger.LogDebug(ctx.Context, "Writing SBOMs sequentially", "bucketName", s3cfg.BucketName, "prefix", s3cfg.Prefix)
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

	// space for proper logging
	fmt.Println()

	for {
		sbom, err := iter.Next(ctx)
		if err == io.EOF {
			break
		}
		sourceAdapter := ctx.Value("source")
		destinationAdapter := ctx.Value("destination")

		var finalProjectName string

		// if the source adapter is local folder cloud storage(s3), and the o/p adapter is local folder or cloud storage(s3),
		// use the SBOM file name as the project name instead of primary comp and version
		// because at the end they have to save the SBOM file as it is.
		if sourceAdapter.(string) == "folder" && destinationAdapter.(string) == "s3" || sourceAdapter.(string) == "s3" && destinationAdapter.(string) == "s3" {
			finalProjectName = sbom.Path
		} else {
			finalProjectName, _ = utils.ConstructProjectName(ctx, "", "", sbom.Namespace, sbom.Version, sbom.Path, sbom.Data, sourceAdapter.(string))
		}

		totalSBOMs++
		if err != nil {
			logger.LogError(ctx.Context, err, "Error retrieving SBOM from iterator")
			continue
		}

		key := filepath.Join(bucketPrefix, finalProjectName)

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
		logger.LogInfo(ctx.Context, "upload", "success", true, "bucket", s3cfg.BucketName, "prefix", s3cfg.Prefix, "filename", finalProjectName)

	}
	logger.LogInfo(ctx.Context, "upload", "total", totalSBOMs, "success", successfullyUploaded, "failed", totalSBOMs-successfullyUploaded)

	return nil
}
