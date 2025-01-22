package interlynk

import (
	"context"
	"fmt"
	"path/filepath"
	"time"
)

// UploadService handles batch uploads of SBOMs to Interlynk
type UploadService struct {
	client        *Client
	maxAttempts   int
	maxConcurrent int
	retryDelay    time.Duration
}

// UploadOptions configures the upload operation
type UploadOptions struct {
	MaxAttempts   int
	MaxConcurrent int
	RetryDelay    time.Duration
}

// NewUploadService creates a new upload service
func NewUploadService(client *Client, opts UploadOptions) *UploadService {
	if opts.MaxAttempts == 0 {
		opts.MaxAttempts = 3
	}
	if opts.MaxConcurrent == 0 {
		opts.MaxConcurrent = 2
	}
	if opts.RetryDelay == 0 {
		opts.RetryDelay = time.Second
	}

	return &UploadService{
		client:        client,
		maxAttempts:   opts.MaxAttempts,
		maxConcurrent: opts.MaxConcurrent,
		retryDelay:    opts.RetryDelay,
	}
}

// UploadResult represents the result of a single SBOM upload
type UploadResult struct {
	Path  string
	Error error
}

// UploadSBOMs uploads multiple SBOM files sequentially
func (s *UploadService) UploadSBOMs(ctx context.Context, files []string) []UploadResult {
	results := make([]UploadResult, len(files))

	for i, file := range files {
		select {
		case <-ctx.Done():
			results[i] = UploadResult{
				Path:  file,
				Error: ctx.Err(),
			}
			return results
		default:
			err := s.client.UploadSBOM(ctx, file)
			results[i] = UploadResult{
				Path:  file,
				Error: err,
			}
		}
	}

	return results
}

// ProcessDirectory uploads all SBOM files from a directory
func (s *UploadService) ProcessDirectory(ctx context.Context, dir string, patterns []string) ([]UploadResult, error) {
	if len(patterns) == 0 {
		patterns = []string{"*.json", "*.xml", "*.yaml", "*.yml", "*.spdx"}
	}

	var files []string
	for _, pattern := range patterns {
		matches, err := filepath.Glob(filepath.Join(dir, pattern))
		if err != nil {
			return nil, fmt.Errorf("glob pattern %s: %w", pattern, err)
		}
		files = append(files, matches...)
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no SBOM files found in %s", dir)
	}

	return s.UploadSBOMs(ctx, files), nil
}
