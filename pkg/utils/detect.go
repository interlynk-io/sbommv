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

package utils

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/interlynk-io/sbommv/pkg/adapters/source"
)

// DetectSourceType determines the InputSource type based on the provided URL or path.
func DetectSourceType(urlStr string) (source.InputSource, error) {
	// Check if it's a valid URL
	u, err := url.Parse(urlStr)
	if err != nil || u.Scheme == "" {
		return DetectLocalSourceType(urlStr), nil
	}

	// Check for specific URL patterns
	host := strings.ToLower(u.Host)

	switch {
	case strings.Contains(host, "github.com"):
		return source.SourceGithub, nil

	case strings.Contains(host, "interlynk.io") || strings.Contains(urlStr, "lynkapi"):
		return source.SourceInterlynk, nil

	case strings.Contains(host, "dependencytrack.com"):
		return source.SourceDependencyTrack, nil

	case strings.Contains(host, "amazonaws.com") || strings.HasPrefix(urlStr, "s3://"):
		return source.SourceS3, nil
	}

	return "", fmt.Errorf("unknown source type for URL: %s", urlStr)
}

// DetectLocalSourceType determines if a local path is a file or directory
func DetectLocalSourceType(path string) source.InputSource {
	info, err := os.Stat(path)
	if err != nil {
		return ""
	}

	if info.IsDir() {
		return source.SourceFolder
	}
	return source.SourceFile
}
