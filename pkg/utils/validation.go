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
	"strings"

	"github.com/interlynk-io/sbommv/pkg/types"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// FlagValidation validates that each adapter should contain flag of respective adapters only
// if a adapter "X" of type Input(in)/Output(out),
// then the flag name should be of the form "out-X-<flag-name>" or "in-X-<flag-name>"
// where X is the adapter name
func FlagValidation(cmd *cobra.Command, adapter types.AdapterType, adapterPrefix types.FlagPrefix) error {
	var err error
	cmd.Flags().Visit(func(f *pflag.Flag) {
		// out-
		flagPrefix := fmt.Sprintf("%s"+"-", string(adapterPrefix))

		// out-folder-
		flagType := fmt.Sprintf("%s%s-", flagPrefix, string(adapter))

		// f.Name: out-interlynk-url

		if strings.HasPrefix(f.Name, flagPrefix) && !strings.HasPrefix(f.Name, flagType) {
			// logger.LogError(cmd.Context(), fmt.Errorf("invalid flag for %s adapter 'github': %s", string(adapterPrefix)+"put", f.Name), "flag", f.Name)
			err = fmt.Errorf("Error: flag --%s is invalid for %s adapter %s", f.Name, string(adapterPrefix)+"put", string(adapter))
		}
	})
	return err
}
