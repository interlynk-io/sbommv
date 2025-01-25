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
	"path/filepath"
	"strings"
)

// validateAddressString validates if the provided string is either a valid URL or file path.
// Returns an error if the string is neither a valid URL nor a valid file path.
func validateAddressString(addr string) error {
	// First try to parse as URL
	if u, err := url.Parse(addr); err == nil && u.Scheme != "" {
		// Valid URL with scheme
		return validateURL(u)
	}

	// If not a URL, validate as file path
	return validatePath(addr)
}

// validateURL performs additional validation on URLs
func validateURL(u *url.URL) error {
	switch strings.ToLower(u.Scheme) {
	case "http", "https":
		if u.Host == "" {
			return fmt.Errorf("missing host in URL: %s", u.String())
		}
		return nil
	case "file":
		return validatePath(u.Path)
	case "s3":
		if u.Host == "" {
			return fmt.Errorf("missing bucket name in S3 URL: %s", u.String())
		}
		return nil
	default:
		return fmt.Errorf("unsupported URL scheme: %s", u.Scheme)
	}
}

// validatePath checks if a path is valid and accessible
func validatePath(path string) error {
	// Clean the path to handle . and .. segments
	path = filepath.Clean(path)

	// Check if path is absolute and exists
	if filepath.IsAbs(path) {
		return validateExistingPath(path)
	}

	// For relative paths, convert to absolute
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("invalid relative path: %w", err)
	}
	return validateExistingPath(absPath)
}

// validateExistingPath checks if a path exists and is accessible
func validateExistingPath(path string) error {
	// Check if path exists
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Special case: if this is a target path for writing,
			// check if the parent directory exists and is writable
			parentDir := filepath.Dir(path)
			parentInfo, parentErr := os.Stat(parentDir)
			if parentErr == nil && parentInfo.IsDir() {
				return nil // Parent directory exists, which is good enough for a target path
			}
			return fmt.Errorf("path does not exist: %s", path)
		}
		if os.IsPermission(err) {
			return fmt.Errorf("permission denied accessing path: %s", path)
		}
		return fmt.Errorf("error accessing path: %w", err)
	}

	// Check if it's a directory and accessible
	if info.IsDir() {
		// Try to read directory to verify permissions
		f, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("cannot access directory: %w", err)
		}
		f.Close()
		return nil
	}

	return nil
}

// ValidateURLs validates both source and target address strings
func ValidateURLs(fromURL, toURL string) error {
	// Validate source
	if err := validateAddressString(fromURL); err != nil {
		return fmt.Errorf("invalid source address: %w", err)
	}

	// Validate target
	if err := validateAddressString(toURL); err != nil {
		return fmt.Errorf("invalid target address: %w", err)
	}

	return nil
}
