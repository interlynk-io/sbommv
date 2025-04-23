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

func (u *S3ParallelUploader) Upload(ctx tcontext.TransferMetadata, s3cfg *S3Config, iter iterator.SBOMIterator) error {
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
