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

package folder

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/google/uuid"
	"github.com/interlynk-io/sbommv/pkg/iterator"
	"github.com/interlynk-io/sbommv/pkg/logger"
	"github.com/interlynk-io/sbommv/pkg/tcontext"
	"github.com/interlynk-io/sbommv/pkg/types"
)

type SBOMUploader interface {
	Upload(ctx *tcontext.TransferMetadata, config *FolderConfig, iter iterator.SBOMIterator) error
}

var uploaderFactory = map[types.UploadMode]SBOMUploader{
	types.UploadSequential: &SequentialUploader{},
	types.UploadParallel:   &ParallelUploader{}, // Add parallel uploader later
}

type SequentialUploader struct{}

// Upload: sequentially, one at a time
func (u *SequentialUploader) Upload(ctx *tcontext.TransferMetadata, config *FolderConfig, iter iterator.SBOMIterator) error {
	logger.LogDebug(ctx.Context, "Writing SBOMs sequentially", "folder", config.FolderPath)
	for {
		sbom, err := iter.Next(*ctx)
		if err == io.EOF {
			break
		}
		if err != nil {
			logger.LogError(ctx.Context, err, "Error retrieving SBOM from iterator")
			return err
		}

		namespace := filepath.Base(sbom.Namespace)
		if namespace == "" {
			namespace = fmt.Sprintf("sbom_%s.json", uuid.New().String())
		}
		outputDir := filepath.Join(config.FolderPath, namespace)
		if err := os.MkdirAll(outputDir, 0o755); err != nil {
			logger.LogError(ctx.Context, err, "Failed to create folder", "path", outputDir)
			return err
		}
		outputFile := filepath.Join(outputDir, sbom.Path)
		if sbom.Path == "" {
			outputFile = filepath.Join(outputDir, fmt.Sprintf("%s.sbom.json", uuid.New().String()))
		}
		if err := os.WriteFile(outputFile, sbom.Data, 0o644); err != nil {
			logger.LogError(ctx.Context, err, "Failed to write SBOM file", "path", outputFile)
			return err
		}
		logger.LogDebug(ctx.Context, "Successfully written SBOM", "path", outputFile)
	}
	return nil
}

type ParallelUploader struct {
	MaxWorkers int
}

func (u *ParallelUploader) Upload(ctx *tcontext.TransferMetadata, config *FolderConfig, iter iterator.SBOMIterator) error {
	logger.LogDebug(ctx.Context, "Writing SBOMs in parallel", "folder", config.FolderPath)

	u.MaxWorkers = 3
	taskChan := make(chan *iterator.SBOM)
	var wg sync.WaitGroup
	errChan := make(chan error, u.MaxWorkers)

	for i := 0; i < u.MaxWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for sbom := range taskChan {
				if err := u.processSBOM(ctx, config, sbom); err != nil {
					errChan <- err
				}
			}
		}()
	}
	for {
		sbom, err := iter.Next(*ctx)
		if err == io.EOF {
			break
		}
		if err != nil {
			logger.LogError(ctx.Context, err, "Error retrieving SBOM from iterator")
			continue
		}
		taskChan <- sbom
	}
	close(taskChan)

	wg.Wait()
	close(errChan)

	for err := range errChan {
		logger.LogError(ctx.Context, err, "Error during parallel upload")
	}

	logger.LogDebug(ctx.Context, "Successfully Writen SBOMs in parallel", "folder", config.FolderPath)

	return nil
}

func (u *ParallelUploader) processSBOM(ctx *tcontext.TransferMetadata, config *FolderConfig, sbom *iterator.SBOM) error {
	namespace := filepath.Base(sbom.Namespace)
	if namespace == "" {
		namespace = fmt.Sprintf("sbom_%s.json", uuid.New().String())
	}
	outputDir := filepath.Join(config.FolderPath, namespace)
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("failed to create folder %s: %w", outputDir, err)
	}
	outputFile := filepath.Join(outputDir, sbom.Path)
	if sbom.Path == "" {
		outputFile = filepath.Join(outputDir, fmt.Sprintf("%s.sbom.json", uuid.New().String()))
	}
	if err := os.WriteFile(outputFile, sbom.Data, 0o644); err != nil {
		return fmt.Errorf("failed to write SBOM file %s: %w", outputFile, err)
	}
	logger.LogDebug(ctx.Context, "Successfully written SBOM", "path", outputFile)
	return nil
}
