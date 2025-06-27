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
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/interlynk-io/sbommv/pkg/logger"
	"github.com/interlynk-io/sbommv/pkg/tcontext"
	_ "modernc.org/sqlite"
)

const (
	CACHE_PATH = ".sbommv/cache.db"
)

// Cache holds in-memory cache data (JSON-like maps) to reduce SQLite queries, synced to cache.db.
type Cache struct {
	Data map[string]AdapterCache
	db   *sql.DB
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

// CachePath generates a daemon-specific cache file path
func CachePath(outputAdapter, method string) string {
	return filepath.Join(".sbommv", fmt.Sprintf("cache_%s_%s.db", outputAdapter, method))
}

const createReposAndSBOMsTable string = `
	CREATE TABLE IF NOT EXISTS repos (
		output_adapter TEXT,
		input_adapter TEXT,
		method TEXT,
		repo TEXT,
		published_at TEXT,
		release_id TEXT,
		PRIMARY KEY (output_adapter, input_adapter, method, repo)
	);

	CREATE TABLE IF NOT EXISTS sboms (
		output_adapter TEXT,
		input_adapter TEXT,
		method TEXT,
		owner TEXT,
		repo TEXT,
		tag_name TEXT,
		filename TEXT,
		processed BOOLEAN,
		PRIMARY KEY (output_adapter, input_adapter, method, repo, tag_name, filename)
	);
`

// InitCache initializes SQLite database with repos and sboms tables.
func (c *Cache) InitCache(ctx tcontext.TransferMetadata, outputAdapter, method string) error {
	path := CachePath(outputAdapter, method)

	logger.LogDebug(ctx.Context, "Initializing SQLite cache", "path", path)

	// Create cache directory
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		logger.LogError(ctx.Context, err, "Failed to create cache directory")
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Open embedded SQLite database with timeout
	dbCtx, cancel := context.WithTimeout(ctx.Context, 5*time.Second)
	defer cancel()

	db, err := sql.Open("sqlite", path)
	if err != nil {
		logger.LogError(ctx.Context, err, "Failed to open SQLite database")
		return fmt.Errorf("failed to open SQLite database: %w", err)
	}

	c.db = db
	logger.LogDebug(ctx.Context, "Cache database opened", "path", path)

	// // Enable WAL mode for concurrent reads/writes
	// if _, err = db.ExecContext(dbCtx, "PRAGMA journal_mode=WAL;"); err != nil {
	// 	logger.LogError(ctx.Context, err, "Failed to enable WAL mode")
	// 	return fmt.Errorf("failed to enable WAL mode: %w", err)
	// }

	// Create tables
	if _, err = db.ExecContext(dbCtx, createReposAndSBOMsTable); err != nil {
		logger.LogError(ctx.Context, err, "Failed to create tables")
		return fmt.Errorf("failed to create tables: %w", err)
	}

	logger.LogDebug(ctx.Context, "Successfully initialized SQLite cache", "path", path)
	return nil
}

// ensureCachePathFor initializes a specific in-memory cache path (DRY helper).
func (c *Cache) ensureCachePathFor(outputAdapter, inputAdapter, method string) {
	if _, exists := c.Data[outputAdapter]; !exists {
		c.Data[outputAdapter] = make(AdapterCache)
	}
	if _, exists := c.Data[outputAdapter][inputAdapter]; !exists {
		c.Data[outputAdapter][inputAdapter] = make(GitHubDaemonCache)
	}
	if _, exists := c.Data[outputAdapter][inputAdapter][method]; !exists {
		c.Data[outputAdapter][inputAdapter][method] = MethodCache{
			Repos: make(map[string]RepoState),
			SBOMs: make(map[string]bool),
		}
	}
}

// LoadCache populates in-memory cache (cache-aside pattern) from SQLite to reduce query frequency.
func (c *Cache) LoadCache(ctx tcontext.TransferMetadata, adapter, method string) error {
	path := CachePath(adapter, method)

	logger.LogDebug(ctx.Context, "Loading cache from SQLite database", "path", path)

	if c.db == nil {
		if err := c.InitCache(ctx, adapter, method); err != nil {
			return err
		}
	}

	c.Lock()
	defer c.Unlock()
	c.Data = make(map[string]AdapterCache)

	rows, err := c.db.Query(`
		SELECT output_adapter, input_adapter, method, repo, published_at, release_id
		FROM repos
	`)
	if err != nil {
		logger.LogError(ctx.Context, err, "Failed to query repos")
		return fmt.Errorf("failed to query repos: %w", err)
	}

	defer rows.Close()

	for rows.Next() {
		var outputAdapter, inputAdapter, method, repo, publishedAt, releaseID string
		if err := rows.Scan(&outputAdapter, &inputAdapter, &method, &repo, &publishedAt, &releaseID); err != nil {
			return fmt.Errorf("failed to scan row: %w", err)
		}
		c.ensureCachePathFor(outputAdapter, inputAdapter, method)

		c.Data[outputAdapter][inputAdapter][method].Repos[repo] = RepoState{
			PublishedAt: publishedAt,
			ReleaseID:   releaseID,
		}
	}

	sbomsRows, err := c.db.Query(`
		SELECT output_adapter, input_adapter, method, owner, repo, tag_name, filename, processed
		FROM sboms
	`)
	if err != nil {
		return fmt.Errorf("failed to query sboms: %w", err)
	}

	defer sbomsRows.Close()

	for sbomsRows.Next() {
		var outputAdapter, inputAdapter, method, owner, repo, tagName, filename string
		var processed bool
		if err := sbomsRows.Scan(&outputAdapter, &inputAdapter, &method, &owner, &repo, &tagName, &filename, &processed); err != nil {
			return fmt.Errorf("failed to scan sboms row: %w", err)
		}

		c.ensureCachePathFor(outputAdapter, inputAdapter, method)
		sbomKey := fmt.Sprintf("%s:%s:%s:%s", owner, repo, tagName, filename)
		c.Data[outputAdapter][inputAdapter][method].SBOMs[sbomKey] = processed
	}

	logger.LogDebug(ctx.Context, "Successfully loaded cache from SQLite", "path", path)
	return nil
}

// SaveCache updates SQLite with in-memory cache changes (write-through caching).
func (c *Cache) SaveCache(ctx tcontext.TransferMetadata, adapter, method string) error {
	path := CachePath(adapter, method)

	if c.db == nil {
		return fmt.Errorf("SQLite database not initialized")
	}
	logger.LogDebug(ctx.Context, "Saving cache to SQLite database", "path", CACHE_PATH)

	// Retry on "database is locked" errors
	for retries := 3; retries > 0; retries-- {
		tx, err := c.db.Begin()
		if err != nil {
			return fmt.Errorf("failed to start transaction: %w", err)
		}

		// Defer rollback in case of failure
		rollback := true
		defer func() {
			if rollback {
				tx.Rollback()
			}
		}()

		for outputAdapter, adapterCache := range c.Data {
			for inputAdapter, daemonCache := range adapterCache {
				for method, methodCache := range daemonCache {
					for repo, state := range methodCache.Repos {
						_, err := tx.Exec(`
							INSERT OR REPLACE INTO repos (output_adapter, input_adapter, method, repo, published_at, release_id)
							VALUES (?, ?, ?, ?, ?, ?)`, outputAdapter, inputAdapter, method, repo, state.PublishedAt, state.ReleaseID)
						if err != nil {
							return fmt.Errorf("failed to save repos: %w", err)
						}
					}

					for sbomKey, processed := range methodCache.SBOMs {
						parts := strings.SplitN(sbomKey, ":", 4)
						if len(parts) != 4 {
							continue // Skip invalid keys
						}

						owner, repo, tagName, filename := parts[0], parts[1], parts[2], parts[3]
						_, err := tx.Exec(`
							INSERT OR REPLACE INTO sboms (output_adapter, input_adapter, method, owner, repo, tag_name, filename, processed)
							VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, outputAdapter, inputAdapter, method, owner, repo, tagName, filename, processed)
						if err != nil {
							return fmt.Errorf("failed to save sboms: %w", err)
						}
					}
				}
			}
		}

		if err := tx.Commit(); err != nil {
			if strings.Contains(err.Error(), "database is locked") && retries > 1 {
				logger.LogDebug(ctx.Context, "Database locked, retrying", "retries_left", retries-1)
				time.Sleep(100 * time.Millisecond)
				continue
			}
			return fmt.Errorf("failed to commit transaction: %w", err)
		}
		rollback = false
		break

	}

	logger.LogDebug(ctx.Context, "Successfully saved cache to SQLite", "path", path)
	return nil
}

func (c *Cache) EnsureCachePath(ctx tcontext.TransferMetadata, outputAdapter, inputAdapter string) {
	logger.LogDebug(ctx.Context, "Ensuring cache path exists", "output_adapter", outputAdapter, "input_adapter", inputAdapter)
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
	logger.LogDebug(ctx.Context, "Initialized cache paths", "output_adapter", outputAdapter, "input_adapter", inputAdapter, "methods", []string{"release", "api", "tool"})
}

// IsSBOMProcessed checks if an SBOM is processed in the cache or not
func (c *Cache) IsSBOMProcessed(ctx tcontext.TransferMetadata, outputAdapter, inputAdapter, method, sbomCacheKey, repo string) bool {
	logger.LogDebug(ctx.Context, "Checking if SBOM is processed", "cache_key", sbomCacheKey, "method", method)

	if c.db == nil {
		return false
	}

	parts := strings.SplitN(sbomCacheKey, ":", 4)
	if len(parts) != 4 {
		return false
	}

	owner, repo, tagName, filename := parts[0], parts[1], parts[2], parts[3]

	var processed bool
	err := c.db.QueryRow(`
		SELECT processed FROM sboms
		WHERE output_adapter = ? AND input_adapter = ? AND method = ? AND owner = ? AND repo = ? AND tag_name = ? AND filename = ?`,
		outputAdapter, inputAdapter, method, owner, repo, tagName, filename).Scan(&processed)

	if err == sql.ErrNoRows {
		return false
	}

	if err != nil {
		logger.LogError(ctx.Context, err, "Failed to check SBOM processed")
		return false
	}

	if processed {
		logger.LogDebug(ctx.Context, "SBOM already processed", "cache_key", sbomCacheKey, "method", method)
	}

	return processed
}

// MarkSBOMProcessed marks an SBOM as processed in the cache (write-through).
func (c *Cache) MarkSBOMProcessed(ctx tcontext.TransferMetadata, outputAdapter, inputAdapter, method, sbomCacheKey, repo string) error {
	if c.db == nil {
		return fmt.Errorf("SQLite database not initialized")
	}

	parts := strings.SplitN(sbomCacheKey, ":", 4)
	if len(parts) != 4 {
		return fmt.Errorf("invalid sbom cache key: %s", sbomCacheKey)
	}

	owner, repo, tagName, filename := parts[0], parts[1], parts[2], parts[3]

	// Retry on "database is locked" errors
	for retries := 3; retries > 0; retries-- {
		tx, err := c.db.Begin()
		if err != nil {
			return fmt.Errorf("failed to start transaction: %w", err)
		}

		rollback := true
		defer func() {
			if rollback {
				tx.Rollback()
			}
		}()

		_, err = tx.Exec(`
			INSERT OR REPLACE INTO sboms (output_adapter, input_adapter, method, owner, repo, tag_name, filename, processed)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			outputAdapter, inputAdapter, method, owner, repo, tagName, filename, true)
		if err != nil {
			return fmt.Errorf("failed to mark SBOM processed: %w", err)
		}

		if err := tx.Commit(); err != nil {
			if strings.Contains(err.Error(), "database is locked") && retries > 1 {
				logger.LogDebug(ctx.Context, "Database locked, retrying", "retries_left", retries-1)
				time.Sleep(100 * time.Millisecond)
				continue
			}
			return fmt.Errorf("failed to commit transaction: %w", err)
		}
		rollback = false
		break
	}

	logger.LogDebug(ctx.Context, "Marked SBOM as processed", "cache_key", sbomCacheKey, "method", method)
	return nil
}

// PruneSBOMs clears SBOMs for a specific adapter, input adapter, method, and repo.
func (c *Cache) PruneSBOMs(ctx tcontext.TransferMetadata, outputAdapter, inputAdapter, method, repo string) error {
	if c.db == nil {
		return fmt.Errorf("SQLite database not initialized")
	}

	tx, err := c.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
		DELETE FROM sboms
		WHERE output_adapter = ? AND input_adapter = ? AND method = ? AND repo = ?`,
		outputAdapter, inputAdapter, method, repo)
	if err != nil {
		return fmt.Errorf("failed to prune SBOMs: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	logger.LogDebug(ctx.Context, "Cleared old SBOMs", "output_adapter", outputAdapter, "method", method, "repo", repo)
	return nil
}
