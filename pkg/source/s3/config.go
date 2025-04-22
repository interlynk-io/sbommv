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
// ------------------

package s3

import (
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/interlynk-io/sbommv/pkg/logger"
	"github.com/interlynk-io/sbommv/pkg/tcontext"
	"github.com/interlynk-io/sbommv/pkg/types"
)

type S3Config struct {
	AccessKey      string
	SecretKey      string
	BucketName     string
	Region         string
	Prefix         string
	ProcessingMode types.ProcessingMode
}

func NewS3Config() *S3Config {
	return &S3Config{
		ProcessingMode: types.FetchSequential, // Default
	}
}

func (s *S3Config) GetBucketName() string {
	return s.BucketName
}

func (s *S3Config) GetRegion() string {
	return s.Region
}

func (s *S3Config) GetPrefix() string {
	return s.Prefix
}

func (s *S3Config) GetProcessingMode() types.ProcessingMode {
	return s.ProcessingMode
}

func (s *S3Config) SetBucketName(bucketName string) {
	s.BucketName = bucketName
}

func (s *S3Config) SetRegion(region string) {
	s.Region = region
}

func (s *S3Config) SetPrefix(prefix string) {
	s.Prefix = prefix
}

func (s *S3Config) SetProcessingMode(mode types.ProcessingMode) {
	s.ProcessingMode = mode
}

func (s *S3Config) SetSecretKey(secretKey string) {
	s.SecretKey = secretKey
}

func (s *S3Config) SetAccessKey(accessKey string) {
	s.AccessKey = accessKey
}

func (s *S3Config) GetAWSClient(ctx tcontext.TransferMetadata) (*s3.Client, error) {
	logger.LogDebug(ctx.Context, "Initializing AWS S3 client", "region", s.Region, "bucket", s.BucketName, "prefix", s.Prefix)

	// Load AWS config
	var cfg aws.Config
	var err error
	if s.AccessKey != "" && s.SecretKey != "" {
		fmt.Println("Using AWS credentials from flags")
		creds := aws.Credentials{
			AccessKeyID:     s.AccessKey,
			SecretAccessKey: s.SecretKey,
		}
		cfg, err = config.LoadDefaultConfig(ctx.Context,
			config.WithRegion(s.Region),
			config.WithCredentialsProvider(aws.NewCredentialsCache(credentials.StaticCredentialsProvider{Value: creds})),
		)
	} else {
		cfg, err = config.LoadDefaultConfig(ctx.Context, config.WithRegion(s.Region))
	}

	if err != nil {
		logger.LogError(ctx.Context, err, "Failed to load AWS config")
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create S3 client
	return s3.NewFromConfig(cfg), err
}
