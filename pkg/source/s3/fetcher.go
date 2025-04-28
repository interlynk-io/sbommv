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
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
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

func (s *S3ParallelFetcher) Fetch(ctx tcontext.TransferMetadata, s3cfg *S3Config) (iterator.SBOMIterator, error) {
	logger.LogDebug(ctx.Context, "Fetching SBOMs Concurrently...")

	bucketPrefix := s3cfg.Prefix

	client, err := s3cfg.GetAWSClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// add "/" to prefix if not present in the end
	if bucketPrefix != "" && !strings.HasSuffix(bucketPrefix, "/") {
		bucketPrefix = bucketPrefix + "/"
	}

	// Validate bucket
	_, err = client.HeadBucket(ctx.Context, &s3.HeadBucketInput{
		Bucket: aws.String(s3cfg.BucketName),
	})
	if err != nil {
		if strings.Contains(err.Error(), "NotFound") || strings.Contains(err.Error(), "NoSuchBucket") || strings.Contains(err.Error(), "404") {
			return nil, fmt.Errorf("bucket %q does not exist", s3cfg.BucketName)
		}
		return nil, fmt.Errorf("failed to access bucket %q: %w", s3cfg.BucketName, err)
	}

	// List objects (single call, no pagination)
	resp, err := client.ListObjectsV2(ctx.Context, &s3.ListObjectsV2Input{
		Bucket: aws.String(s3cfg.BucketName),
		Prefix: aws.String(s3cfg.Prefix),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list objects: %w", err)
	}

	var sboms []*iterator.SBOM
	var mu sync.Mutex
	var wg sync.WaitGroup
	const maxConcurrency = 3
	semaphore := make(chan struct{}, maxConcurrency)

	for _, obj := range resp.Contents {
		wg.Add(1)
		semaphore <- struct{}{}
		go func(key string) {
			defer wg.Done()
			defer func() { <-semaphore }()

			// Download object
			getResp, err := client.GetObject(ctx.Context, &s3.GetObjectInput{
				Bucket: aws.String(s3cfg.BucketName),
				Key:    aws.String(key),
			})
			if err != nil {
				logger.LogDebug(ctx.Context, "Failed to download", "key", key, "error", err)
				return
			}

			// Read content
			content, err := io.ReadAll(getResp.Body)
			getResp.Body.Close()
			if err != nil {
				logger.LogDebug(ctx.Context, "Failed to read", "key", key, "error", err)
				return
			}

			// Validate SBOM
			if !source.IsSBOMFile(content) {
				logger.LogDebug(ctx.Context, "Skipping invalid SBOM", "key", key)
				return
			}

			// Store SBOM
			mu.Lock()
			sboms = append(sboms, &iterator.SBOM{
				Path:      strings.TrimPrefix(*obj.Key, *resp.Prefix),
				Data:      content,
				Namespace: s3cfg.BucketName + "-" + s3cfg.Prefix,
			})
			mu.Unlock()
			logger.LogDebug(ctx.Context, "Fetched SBOM", "key", key, "size", len(content))
		}(*obj.Key)
	}

	wg.Wait()

	if len(sboms) == 0 {
		return nil, fmt.Errorf("no SBOMs found in s3://%s/%s", s3cfg.BucketName, s3cfg.Prefix)
	}

	return NewS3Iterator(sboms), nil
}

// Fetching SBOMs from S3 bucket sequentially
func (s *S3SequentialFetcher) Fetch(ctx tcontext.TransferMetadata, s3cfg *S3Config) (iterator.SBOMIterator, error) {
	logger.LogDebug(ctx.Context, "Fetching SBOMs Sequentially")
	bucketPrefix := s3cfg.Prefix

	client, err := s3cfg.GetAWSClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// add "/" to prefix if not present in the end
	if bucketPrefix != "" && !strings.HasSuffix(bucketPrefix, "/") {
		bucketPrefix = bucketPrefix + "/"
	}

	// Validate bucket
	_, err = client.HeadBucket(ctx.Context, &s3.HeadBucketInput{Bucket: aws.String(s3cfg.BucketName)})
	if err != nil {
		if strings.Contains(err.Error(), "NotFound") || strings.Contains(err.Error(), "NoSuchBucket") || strings.Contains(err.Error(), "404") {
			return nil, fmt.Errorf("bucket %q does not exist", s3cfg.BucketName)
		}
		return nil, fmt.Errorf("failed to access bucket %q: %w", s3cfg.BucketName, err)
	}

	logger.LogDebug(ctx.Context, "Fetching SBOMs from S3 bucket", "bucket", s3cfg.BucketName, "prefix", s3cfg.Prefix, "region", s3cfg.Region)

	// List objects
	resp, err := client.ListObjectsV2(ctx.Context, &s3.ListObjectsV2Input{
		Bucket: aws.String(s3cfg.BucketName),
		Prefix: aws.String(bucketPrefix),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list objects: %w", err)
	}

	// Process objects
	var sbomList []*iterator.SBOM
	for _, obj := range resp.Contents {

		// Download object
		getResp, err := client.GetObject(ctx.Context, &s3.GetObjectInput{
			Bucket: aws.String(s3cfg.BucketName),
			Key:    obj.Key,
		})
		if err != nil {
			logger.LogDebug(ctx.Context, "Failed to download", "key", *obj.Key, "error", err)
			continue
		}
		logger.LogDebug(ctx.Context, "Get Object Response", "content_length", getResp.ContentLength, "content_type", getResp.ContentType)

		content, err := io.ReadAll(getResp.Body)
		getResp.Body.Close()
		if err != nil {
			logger.LogDebug(ctx.Context, "Failed to read", "key", *obj.Key, "error", err)
			getResp.Body.Close()
			continue
		}
		getResp.Body.Close()

		// check whether it's a SBOM content or not
		if !source.IsSBOMFile(content) {
			logger.LogDebug(ctx.Context, "Skipping invalid SBOM", "key", *obj.Key, "content_sample", string(content[:min(100, len(content))]))
			continue
		}

		sbomList = append(sbomList, &iterator.SBOM{
			Path:      strings.TrimPrefix(*obj.Key, *resp.Prefix),
			Data:      content,
			Namespace: s3cfg.BucketName + "-" + s3cfg.Prefix,
		})
		logger.LogDebug(ctx.Context, "Fetched SBOM", "key", *obj.Key, "size", len(content))

	}

	if len(sbomList) == 0 {
		return nil, fmt.Errorf("no SBOMs found in s3://%s/%s", s3cfg.BucketName, s3cfg.Prefix)
	}
	return NewS3Iterator(sbomList), nil
}
