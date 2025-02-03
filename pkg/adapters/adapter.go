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
	"context"
	"fmt"

	"github.com/interlynk-io/sbommv/pkg/adapters/dest"
	"github.com/interlynk-io/sbommv/pkg/adapters/source"
	"github.com/interlynk-io/sbommv/pkg/logger"
	"github.com/interlynk-io/sbommv/pkg/mvtypes"
)

// NewSourceAdapter creates an appropriate adapter for the provided source type.
// It returns an error if the source type is not supported or if required
// configuration is missing.
func NewSourceAdapter(ctx context.Context, config mvtypes.Config) (source.InputAdapter, error) {
	logger.LogInfo(ctx, "Initializing source adapter", "sourceType", config.SourceType)

	var adapter source.InputAdapter
	var err error

	switch source.InputType(config.SourceType) {

	case source.SourceGithub:
		adapter = source.NewGitHubAdapter(config)

	default:
		err = fmt.Errorf("unsupported input source type: %s", string(config.SourceType))
		logger.LogError(ctx, err, "Invalid source adapter type provided")
		return nil, err
	}

	logger.LogDebug(ctx, "Successfully initialized source adapter", "adapterType", config.SourceType)
	return adapter, nil
}

// NewOutputAdapter creates an appropriate output adapter for the given type
func NewDestAdapter(ctx context.Context, config mvtypes.Config) (dest.OutputAdapter, error) {
	logger.LogInfo(ctx, "Initializing destination adapter", "destinationType", config.DestinationType)

	var adapter dest.OutputAdapter
	var err error

	switch dest.OutputType(config.DestinationType) {

	case dest.DestInterlynk:
		adapter = dest.NewInterlynkAdapter(config)

	default:
		err = fmt.Errorf("unsupported input destination type: %s", string(config.DestinationType))
		logger.LogError(ctx, err, "Invalid destination adapter type provided")
		return nil, err
	}

	logger.LogDebug(ctx, "Successfully initialized destination adapter", "adapterType", config.DestinationType)
	return adapter, nil
}
