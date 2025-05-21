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

package github

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/interlynk-io/sbommv/pkg/logger"
	"github.com/interlynk-io/sbommv/pkg/tcontext"
)

const (
	CACHE_PATH = ".sbommv/cache.json"
)

type GithubWatcherFetcher struct{}

func NewWatcherFetcher() *GithubWatcherFetcher {
	return &GithubWatcherFetcher{}
}

type Cache struct {
	Data map[string]AdapterCache `json:"data"`
	sync.RWMutex
}

type AdapterCache map[string]GitHubDaemonCache

type GitHubDaemonCache map[string]MethodCache

type MethodCache struct {
	Repos map[string]RepoState `json:"repos"`
	SBOMs map[string]bool      `json:"sboms"`
}

// RepoState stores release information.
type RepoState struct {
	PublishedAt string `json:"published_at"`
	ReleaseID   string `json:"release_id"`
}

// NewCache initializes a cache.
func NewCache() *Cache {
	return &Cache{
		Data: make(map[string]AdapterCache),
	}
}

func (c *Cache) LoadCache(ctx tcontext.TransferMetadata, path string) error {
	c.Lock()
	defer c.Unlock()

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		logger.LogDebug(ctx.Context, "Cache file does not exist, starting with empty cache", "path", path)
		return nil // Cache doesn't exist
	}

	if err != nil {
		return fmt.Errorf("failed to read cache file: %w", err)
	}

	if err := json.Unmarshal(data, c); err != nil {
		logger.LogDebug(ctx.Context, "Failed to parse cache file, starting with empty cache", "path", path, "error", err)
		return nil // Invalid cache
	}

	logger.LogDebug(ctx.Context, "Successfully loaded cache", "path", path)
	return nil
}

// SaveCache writes the cache to file.
func (c *Cache) SaveCache(ctx tcontext.TransferMetadata, path string) error {
	logger.LogDebug(ctx.Context, "Saving cache to file", "path", path)
	c.RLock()
	defer c.RUnlock()

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cache: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	logger.LogDebug(ctx.Context, "Successfully saved cache", "path", path)

	return nil
}

func (c *Cache) EnsureCachePath(ctx tcontext.TransferMetadata, outputAdapter, inputAdapter string) {
	c.Lock()
	defer c.Unlock()

	if _, exists := c.Data[outputAdapter]; !exists {
		c.Data[outputAdapter] = make(AdapterCache)
	}

	if _, exists := c.Data[outputAdapter][inputAdapter]; !exists {
		c.Data[outputAdapter][inputAdapter] = make(GitHubDaemonCache)
	}

	// intialize all methods
	for _, method := range []string{string(MethodAPI), string(MethodReleases), string(MethodTool)} {
		if _, exists := c.Data[outputAdapter][inputAdapter][method]; !exists {
			c.Data[outputAdapter][inputAdapter][method] = MethodCache{
				Repos: make(map[string]RepoState),
				SBOMs: make(map[string]bool),
			}
		}
	}
	logger.LogDebug(ctx.Context, "Initialized cache paths", "output", outputAdapter, "input", inputAdapter, "methods", []string{"release", "api", "tool"})
}
