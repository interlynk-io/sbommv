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
// -------------------------------------------------------------------------

package adapter

import (
	"fmt"

	"github.com/interlynk-io/sbommv/pkg/adapters/dest"
	"github.com/interlynk-io/sbommv/pkg/adapters/source"
	"github.com/interlynk-io/sbommv/pkg/mvtypes"
)

// NewSourceAdapter creates an appropriate adapter for the given source type.
// It returns an error if the source type is not supported or if required
// configuration is missing.
func NewSourceAdapter(config mvtypes.Config) (source.InputAdapter, error) {
	switch source.InputType(config.SourceType) {

	case source.SourceFile:
		return source.NewFileAdapter(config)

	case source.SourceFolder:
		return source.NewFolderAdapter(config)

	case source.SourceGithub:
		return source.NewGitHubAdapter(config), nil

	case source.SourceS3:
		return source.NewS3Adapter(config)

	case source.SourceInterlynk:
		return source.NewInterlynkAdapter(config), nil

	default:
		return nil, fmt.Errorf("unsupported input source type: %s", string(config.SourceType))
	}
}

// TODO: func NewDestAdapter()

// NewOutputAdapter creates an appropriate output adapter for the given type
func NewDestAdapter(config mvtypes.Config) (dest.OutputAdapter, error) {
	switch dest.OutputType(config.DestinationType) {
	case dest.DestInterlynk:
		return dest.NewInterlynkAdapter(config), nil
		// if config.ProjectID == "" {
		// 	return nil, fmt.Errorf("Interlynk adapter requires project ID")
		// }

	// case dest.DestDependencyTrack:
	// 	if config.ProjectID == "" {
	// 		return nil, fmt.Errorf("DependencyTrack adapter requires project ID")
	// 	}
	// 	return NewDependencyTrackAdapter(
	// 		config.BaseURL,
	// 		config.ProjectID,
	// 		config.APIKey,
	// 		config.InputOptions,
	// 	), nil

	default:
		return nil, fmt.Errorf("unsupported output type: %s", string(config.DestinationType))
	}
}
