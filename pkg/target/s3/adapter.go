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
	"github.com/interlynk-io/sbommv/pkg/utils"
	"github.com/spf13/cobra"
)

type S3Adapter struct {
	Config         *S3Config
	Role           types.AdapterRole
	ProcessingMode types.ProcessingMode
	Uploader       SBOMUploader
}

// AddCommandParams adds S3-specific CLI flags
func (s3 *S3Adapter) AddCommandParams(cmd *cobra.Command) {
	cmd.Flags().String("out-s3-bucket-name", "", "S3 bucket name")
	cmd.Flags().String("out-s3-region", "", "S3 region")
	cmd.Flags().String("out-s3-prefix", "", "S3 prefix")
	cmd.Flags().String("out-s3-access-key", "", "AWS access key for S3")
	cmd.Flags().String("out-s3-secret-key", "", "AWS secret key for S3")
}

// ParseAndValidateParams validates the S3 adapter params
func (s *S3Adapter) ParseAndValidateParams(cmd *cobra.Command) error {
	var (
		bucketNameFlag, regionFlag, prefixFlag, accessKeyFlag, secretKeyFlag string
		missingFlags                                                         []string
		invalidFlags                                                         []string
	)

	bucketNameFlag = "out-s3-bucket-name"
	regionFlag = "out-s3-region"
	prefixFlag = "out-s3-prefix"
	accessKeyFlag = "out-s3-access-key"
	secretKeyFlag = "out-s3-secret-key"

	var bucketName, region, prefix string
	var uploader SBOMUploader

	if s.ProcessingMode == types.ProcessingMode(types.UploadSequential) {
		uploader = &S3SequentialUploader{}
	} else if s.ProcessingMode == types.ProcessingMode(types.UploadParallel) {
		uploader = &S3ParallelUploader{}
	} else {
		return fmt.Errorf("unsupported processing mode: %s", s.ProcessingMode)
	}

	// validate flags for S3 adapter, all flags should start with "in-s3-"
	err := utils.FlagValidation(cmd, types.S3AdapterType, types.OutputAdapterFlagPrefix)
	if err != nil {
		return fmt.Errorf("s3 flag validation failed: %w", err)
	}

	// extract the bucket name
	bucketName, _ = cmd.Flags().GetString(bucketNameFlag)
	if bucketName == "" {
		missingFlags = append(missingFlags, bucketNameFlag)
	}

	// extrack the region name
	region, _ = cmd.Flags().GetString(regionFlag)
	if region == "" {
		// set default as us-east-1
		region = "us-east-1"
	}

	// extract the prefix name
	// if prefix is empty, that means to upload inside bucket itself
	prefix, _ = cmd.Flags().GetString(prefixFlag)

	// extract AWS access Key
	accessKey, _ := cmd.Flags().GetString(accessKeyFlag)

	// extract AWS secret Key
	secretKey, _ := cmd.Flags().GetString(secretKeyFlag)

	if len(missingFlags) > 0 {
		return fmt.Errorf("missing flags: %s", strings.Join(missingFlags, ", "))
	}

	if len(invalidFlags) > 0 {
		return fmt.Errorf("invalid input adapter flag usage:\n %s\n\nUse 'sbommv transfer --help' for correct usage.", strings.Join(invalidFlags, "\n "))
	}

	cfg := NewS3Config()
	cfg.SetProcessingMode(s.ProcessingMode)
	cfg.SetBucketName(bucketName)
	cfg.SetRegion(region)
	cfg.SetPrefix(prefix)
	cfg.SetAccessKey(accessKey)
	cfg.SetSecretKey(secretKey)

	s.Config = cfg
	s.Uploader = uploader

	return nil
}

// FetchSBOMs retrieves SBOMs lazily
func (s *S3Adapter) FetchSBOMs(ctx tcontext.TransferMetadata) (iterator.SBOMIterator, error) {
	return nil, fmt.Errorf("S3 adapter does not support SBOM Fetching when it is in output adapter role")
}

// UploadSBOMs writes SBOMs to the output folder
func (s *S3Adapter) UploadSBOMs(ctx tcontext.TransferMetadata, iter iterator.SBOMIterator) error {
	logger.LogDebug(ctx.Context, "Starting SBOM upload", "mode", s.ProcessingMode)
	return s.Uploader.Upload(ctx, s.Config, iter)
}

// DryRun for Output Adapter: Simulates writing SBOMs to a folder
func (s *S3Adapter) DryRun(ctx tcontext.TransferMetadata, iter iterator.SBOMIterator) error {
	reporter := NewS3Reporter(false, "", s.Config.BucketName, s.Config.Prefix)
	return reporter.DryRun(ctx, iter)
}
