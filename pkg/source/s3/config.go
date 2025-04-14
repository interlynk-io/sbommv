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

import "github.com/interlynk-io/sbommv/pkg/types"

type S3Config struct {
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

// func (c *S3Config) Validate() error {
// 	if c.BucketName == "" {
// 		return types.NewInvalidConfigError("BucketName is required")
// 	}
// 	if c.Region == "" {
// 		return types.NewInvalidConfigError("Region is required")
// 	}
// 	if c.Prefix == "" {
// 		return types.NewInvalidConfigError("Prefix is required")
// 	}
// 	if !isValidProcessingMode(c.ProcessingMode) {
// 		return types.NewInvalidConfigError("Invalid ProcessingMode")
// 	}
// 	return nil
// }

func isValidProcessingMode(mode types.ProcessingMode) bool {
	switch mode {
	case types.FetchSequential, types.FetchParallel:
		return true
	default:
		return false
	}
}
