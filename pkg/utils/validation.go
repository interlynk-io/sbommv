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

	"github.com/interlynk-io/sbommv/pkg/logger"
	"github.com/interlynk-io/sbommv/pkg/types"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// FlagValidation validates that each adapter should contain flag of respective adapters only
func FlagValidation(cmd *cobra.Command, adapter types.AdapterType, adapterPrefix types.FlagPrefix) {
	cmd.Flags().Visit(func(f *pflag.Flag) {
		// out-
		flagPrefix := fmt.Sprintf("%s"+"-", string(adapterPrefix))

		// out-folder-
		flagType := fmt.Sprintf("%s%s-", flagPrefix, string(adapter))

		// f.Name: out-interlynk-url

		if strings.HasPrefix(f.Name, flagPrefix) && !strings.HasPrefix(f.Name, flagType) {
			logger.LogError(cmd.Context(), fmt.Errorf("invalid flag for input adapter 'github': %s", f.Name), "flag", f.Name)
			panic(fmt.Sprintf("Error: flag --%s is invalid for input adapter %s", f.Name, string(adapter)))
		}
	})
}
