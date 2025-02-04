package iterator

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/interlynk-io/sbommv/pkg/logger"
	"github.com/interlynk-io/sbommv/pkg/source/github"
)

// GitHubReleaseIterator iterates over SBOMs fetched from GitHub Releases
type GitHubReleaseIterator struct {
	ctx      context.Context
	client   *github.Client
	sboms    []*SBOM // Store all fetched SBOMs
	position int     // Tracks iteration position
}

// NewGitHubReleaseIterator initializes the iterator and fetches SBOMs at creation
func NewGitHubReleaseIterator(ctx context.Context, client *github.Client) (*GitHubReleaseIterator, error) {
	logger.LogDebug(ctx, "Initializing GitHub Release Iterator", "repo", client.RepoURL)

	// Fetch all SBOMs from GitHub releases
	releaseSBOMs, err := client.FetchSBOMsFromReleases(ctx)
	if err != nil {
		logger.LogError(ctx, err, "Failed to fetch SBOMs from GitHub releases")
		return nil, err
	}
	if len(releaseSBOMs) == 0 {
		return nil, fmt.Errorf("no SBOMs found in repository")
	}
	logger.LogDebug(ctx, "Total SBOMs found in the repository", "total sboms", len(releaseSBOMs))

	// Convert fetched SBOMs into SBOM objects with file storage
	var sboms []*SBOM
	for _, sbomData := range releaseSBOMs {
		sbomFilePath, err := saveSBOMToFile(sbomData)
		if err != nil {
			logger.LogError(ctx, err, "Failed to save SBOM to file")
			continue
		}
		sboms = append(sboms, &SBOM{Path: sbomFilePath, Data: sbomData})
	}

	logger.LogDebug(ctx, "Fetched SBOMs stored successfully", "count", len(sboms))

	// Create and return the iterator
	return &GitHubReleaseIterator{
		ctx:    ctx,
		client: client,
		sboms:  sboms,
	}, nil
}

// Next returns the next SBOM from the stored list
func (it *GitHubReleaseIterator) Next(ctx context.Context) (*SBOM, error) {
	if it.position >= len(it.sboms) {
		return nil, io.EOF // No more SBOMs left
	}

	sbom := it.sboms[it.position]
	it.position++ // Move to the next SBOM
	return sbom, nil
}

// saveSBOMToFile saves the fetched SBOM data into a temporary file
func saveSBOMToFile(sbomData []byte) (string, error) {
	// Create a temporary directory if not exists
	tempDir := os.TempDir()
	sbomFilePath := filepath.Join(tempDir, "github_api_sbom.json")

	// Write SBOM to file
	err := os.WriteFile(sbomFilePath, sbomData, 0o644)
	if err != nil {
		return "", fmt.Errorf("failed to write SBOM to file: %w", err)
	}

	return sbomFilePath, nil
}
