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
	"fmt"
	"strings"

	"github.com/interlynk-io/sbommv/pkg/iterator"
	"github.com/interlynk-io/sbommv/pkg/logger"
	"github.com/interlynk-io/sbommv/pkg/tcontext"
	"github.com/interlynk-io/sbommv/pkg/types"
	"github.com/spf13/cobra"
)

type S3Adapter struct {
	Config         *S3Config
	Role           types.AdapterRole // "input" or "output" adapter type
	ProcessingMode types.ProcessingMode
	Fetcher        SBOMFetcher
}

// AddCommandParams adds S3-specific CLI flags
func (s3 *S3Adapter) AddCommandParams(cmd *cobra.Command) {
	cmd.Flags().String("in-s3-bucket-name", "", "S3 bucket name")
	cmd.Flags().String("in-s3-region", "", "S3 region")
	cmd.Flags().String("in-s3-prefix", "", "S3 prefix")
}

// ParseAndValidateParams validates the S3 adapter params
func (s *S3Adapter) ParseAndValidateParams(cmd *cobra.Command) error {
	var (
		bucketNameFlag, regionFlag, prefixFlag string
		missingFlags                           []string
		invalidFlags                           []string
	)

	bucketNameFlag = "in-s3-bucket-name"
	regionFlag = "in-s3-region"
	prefixFlag = "in-s3-prefix"

	// err := utils.FlagValidation(cmd, types.S3AdapterType, types.InputAdapterFlagPrefix)
	// if err != nil {
	// 	return fmt.Errorf("S3 flag validation failed: %w", err)
	// }

	var bucketName, region, prefix string
	var fetcher SBOMFetcher

	if s.ProcessingMode == types.FetchSequential {
		fetcher = &S3SequentialFetcher{}
	} else if s.ProcessingMode == types.FetchParallel {
		fetcher = &S3ParallelFetcher{}
	} else {
		return fmt.Errorf("unsupported processing mode: %s", s.ProcessingMode)
	}

	// extract the bucket name
	bucketName, _ = cmd.Flags().GetString(bucketNameFlag)
	if bucketName == "" {
		missingFlags = append(missingFlags, bucketNameFlag)
	}

	// extrack the region name
	region, _ = cmd.Flags().GetString(regionFlag)
	if region == "" {
		// missingFlags = append(missingFlags, regionFlag)
	}

	// extract the prefix name
	prefix, _ = cmd.Flags().GetString(prefixFlag)
	if prefix == "" {
		missingFlags = append(missingFlags, prefixFlag)
	}

	if len(missingFlags) > 0 {
		return fmt.Errorf("missing flags: %s", strings.Join(missingFlags, ", "))
	}

	if len(invalidFlags) > 0 {
		return fmt.Errorf("invalid input adapter flag usage:\n %s\n\nUse 'sbommv transfer --help' for correct usage.", strings.Join(invalidFlags, "\n "))
	}

	// cfg := S3Config{
	// 	ProcessingMode: s.ProcessingMode,
	// 	BucketName:     bucketName,
	// 	Region:         region,
	// 	Prefix:         prefix,
	// }
	cfg := NewS3Config()
	cfg.SetProcessingMode(s.ProcessingMode) // Default
	cfg.SetBucketName(bucketName)
	cfg.SetRegion(region)
	cfg.SetPrefix(prefix)
	s.Config = cfg

	fmt.Println("Config:", s.Config)
	s.Fetcher = fetcher

	return nil
}

func (s3 *S3Adapter) FetchSBOMs(ctx tcontext.TransferMetadata) (iterator.SBOMIterator, error) {
	logger.LogDebug(ctx.Context, "Initializing SBOM fetching", "mode", s3.ProcessingMode)
	return s3.Fetcher.Fetch(ctx, s3.Config)
}

func (s3 *S3Adapter) UploadSBOMs(ctx tcontext.TransferMetadata, iterator iterator.SBOMIterator) error {
	return fmt.Errorf("S3 adapter does not support SBOM uploading")
}

func (s3 *S3Adapter) DryRun(ctx tcontext.TransferMetadata, iterator iterator.SBOMIterator) error {
	// Implement dry run logic here
	reporter := NewS3Reporter(false, "", s3.Config.BucketName, s3.Config.Prefix)
	return reporter.DryRun(ctx, iterator)
}
